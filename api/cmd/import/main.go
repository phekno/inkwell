// Command import loads a Notion "Markdown & CSV" export into inkwell, sealing
// each page and writing it directly to DynamoDB.
//
// Usage:
//
//	import <export-dir> --sub <cognito-sub> [--table NAME] [--kms-key-id ID]
//	                    [--region REGION] [--dry-run]
//
// --table/--kms-key-id fall back to $ENTRIES_TABLE/$KMS_KEY_ID. AWS credentials
// come from the standard SDK chain (e.g. AWS_PROFILE=phekno).
package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/kms"

	"github.com/phekno/inkwell/api/internal/crypto"
	"github.com/phekno/inkwell/api/internal/notion"
	"github.com/phekno/inkwell/api/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	sub := fs.String("sub", "", "Cognito sub (user id) to import entries under")
	table := fs.String("table", os.Getenv("ENTRIES_TABLE"), "DynamoDB table name")
	keyID := fs.String("kms-key-id", os.Getenv("KMS_KEY_ID"), "KMS key id or alias")
	region := fs.String("region", os.Getenv("AWS_REGION"), "AWS region")
	dryRun := fs.Bool("dry-run", false, "parse and print the plan without writing")
	include := fs.String("include", "", "comma-separated substrings; undated files matching these are imported anyway (at --undated-date) instead of skipped")
	undatedDate := fs.String("undated-date", "2000-01-01", "date (YYYY-MM-DD) assigned to included undated files")

	// Accept the export dir in either position: before or after the flags.
	args := os.Args[1:]
	var root string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		root, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if root == "" {
		root = fs.Arg(0)
	}
	if root == "" || fs.NArg() > 1 {
		return fmt.Errorf("usage: import <export-dir> --sub <cognito-sub> [flags]")
	}
	root = filepath.Clean(root)

	var includes []string
	for s := range strings.SplitSeq(*include, ",") {
		if s = strings.TrimSpace(s); s != "" {
			includes = append(includes, s)
		}
	}
	fallback, err := time.ParseInLocation("2006-01-02", *undatedDate, time.Local)
	if err != nil {
		return fmt.Errorf("invalid --undated-date %q: %w", *undatedDate, err)
	}

	planned, skipped, err := collect(root, includes, fallback)
	if err != nil {
		return err
	}
	printPlan(planned, skipped)

	if *dryRun {
		fmt.Println("\n(dry run — nothing written)")
		return nil
	}
	if *sub == "" {
		return fmt.Errorf("--sub is required for a real import")
	}
	if *table == "" || *keyID == "" {
		return fmt.Errorf("--table and --kms-key-id (or $ENTRIES_TABLE/$KMS_KEY_ID) are required")
	}

	ctx := context.Background()
	var opts []func(*config.LoadOptions) error
	if *region != "" {
		opts = append(opts, config.WithRegion(*region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("aws config: %w", err)
	}
	seal := &crypto.Sealer{KMS: kms.NewFromConfig(cfg), KeyID: *keyID}
	st := &store.Store{DDB: dynamodb.NewFromConfig(cfg), Table: *table}

	fmt.Printf("\nimporting %d entries under USER#%s …\n", len(planned), *sub)
	for i, p := range planned {
		env, err := seal.Seal(ctx, *sub, []byte(p.Body))
		if err != nil {
			return fmt.Errorf("seal %q: %w", p.Path, err)
		}
		if err := st.Put(ctx, *sub, p.ID, &store.Entry{
			Title:      p.Title,
			Folder:     p.Folder,
			Ciphertext: env.Ciphertext,
			Nonce:      env.Nonce,
			WrappedDEK: env.WrappedDEK,
			CreatedAt:  p.Created,
			UpdatedAt:  p.Updated,
		}); err != nil {
			return fmt.Errorf("put %q: %w", p.Path, err)
		}
		if n := i + 1; n%100 == 0 || n == len(planned) {
			fmt.Printf("  %d/%d\n", n, len(planned))
		}
	}
	fmt.Println("done.")
	return nil
}

type skip struct{ path, reason string }

// collect walks the export tree and resolves every .md file into an import plan
// or a skip. Undated files whose path matches an include substring are imported
// at the fallback date instead of being skipped.
func collect(root string, includes []string, fallback time.Time) ([]notion.Planned, []skip, error) {
	var planned []notion.Planned
	var skipped []skip

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		b, err := os.ReadFile(path) // #nosec G304,G122 -- one-time local import of the user's own export tree
		if err != nil {
			return err
		}
		content := string(b)
		if p, ok, reason := notion.Plan(root, path, content); ok {
			planned = append(planned, p)
		} else if matchesAny(path, includes) {
			p, err := notion.PlanForced(root, path, content, fallback)
			if err != nil {
				return err
			}
			planned = append(planned, p)
		} else {
			skipped = append(skipped, skip{path: path, reason: reason})
		}
		return nil
	})
	return planned, skipped, err
}

func matchesAny(path string, subs []string) bool {
	for _, s := range subs {
		if strings.Contains(path, s) {
			return true
		}
	}
	return false
}

func printPlan(planned []notion.Planned, skipped []skip) {
	byFolder := map[string]int{}
	titleDated := 0
	for _, p := range planned {
		folder := p.Folder
		if folder == "" {
			folder = "(root)"
		}
		byFolder[folder]++
		if p.DateSource == "title" {
			titleDated++
		}
	}

	fmt.Printf("Importable: %d entries (%d dated from title, %d skipped)\n\n",
		len(planned), titleDated, len(skipped))

	folders := make([]string, 0, len(byFolder))
	for f := range byFolder {
		folders = append(folders, f)
	}
	sort.Strings(folders)
	fmt.Println("Per folder:")
	for _, f := range folders {
		fmt.Printf("  %-24s %d\n", f, byFolder[f])
	}

	if titleDated > 0 {
		fmt.Println("\nDated from title:")
		for _, p := range planned {
			if p.DateSource == "title" {
				fmt.Printf("  %s → %s\n", p.Title, p.Created.Format("2006-01-02"))
			}
		}
	}

	var fallbackDated []notion.Planned
	for _, p := range planned {
		if p.DateSource == "fallback" {
			fallbackDated = append(fallbackDated, p)
		}
	}
	if len(fallbackDated) > 0 {
		fmt.Println("\nIncluded with fallback date:")
		for _, p := range fallbackDated {
			fmt.Printf("  %s → %s\n", p.Title, p.Created.Format("2006-01-02"))
		}
	}

	if len(skipped) > 0 {
		fmt.Println("\nSkipped:")
		for _, s := range skipped {
			fmt.Printf("  %s (%s)\n", filepath.Base(s.path), s.reason)
		}
	}
}
