package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/mattn/go-isatty"

	"github.com/dovixman/harness-profiles/internal/adapters/configyaml"
	"github.com/dovixman/harness-profiles/internal/adapters/fsops"
	"github.com/dovixman/harness-profiles/internal/adapters/tui"
	"github.com/dovixman/harness-profiles/internal/app"
	"github.com/dovixman/harness-profiles/internal/domain"
)

var version = "dev"

type commandFunc func(app.Service, []string, io.Writer, io.Writer) int

var commands = map[string]commandFunc{
	"where": func(service app.Service, _ []string, stdout, stderr io.Writer) int {
		return runWhere(service, stdout, stderr)
	},
	"ls": func(service app.Service, _ []string, stdout, stderr io.Writer) int {
		return runList(service, stdout, stderr)
	},
	"tui": func(service app.Service, _ []string, _ io.Writer, stderr io.Writer) int {
		return runTUI(service, stderr)
	},
	"add":    runAdd,
	"update": runUpdate,
	"delete": runDeleteHarness,
	"rm":     runDeleteHarness,
}

var interactiveTerminal = func() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
}

var runTUIProgram = tui.Run
var runTUIFlowProgram = tui.RunFlow

func Run(args []string, stdout, stderr io.Writer) int {
	paths, err := configyaml.DefaultPaths()
	if err != nil {
		writef(stderr, "hp: %v\n", err)
		return 1
	}

	service := app.Service{
		Repo:  configyaml.NewRepository(paths),
		Paths: paths,
		FS:    fsops.OS{},
	}

	if len(args) == 0 {
		if interactiveTerminal() {
			return runTUI(service, stderr)
		}
		printHelp(stdout)
		return 0
	}

	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printHelp(stdout)
		return 0
	}

	command := args[0]
	if command == "--version" || command == "version" {
		writef(stdout, "hp %s\n", version)
		return 0
	}
	if handler, ok := commands[command]; ok {
		return handler(service, args[1:], stdout, stderr)
	}
	return runHarnessCommand(service, args, stdout, stderr)
}

func printHelp(w io.Writer) {
	write(w, `Usage: hp <command>

Commands:
  add         Register a harness
  update      Update a harness
  delete      Delete a harness
	  where       Print config and profiles paths
	  ls          List configured harnesses
	  tui         Open the interactive terminal UI
	  <harness>   Run profile commands: ls, current, switch, adopt, clone, delete, where
  version     Print version
  help        Print help

Global Flags:
  -h, --help  Print help
`)
}

func runTUI(service app.Service, stderr io.Writer) int {
	if err := runTUIProgram(service); err != nil {
		writef(stderr, "hp tui: %v\n", err)
		return 1
	}
	return 0
}

func runTUIFlow(service app.Service, flow tui.Flow, harnessID string, stderr io.Writer) int {
	if err := runTUIFlowProgram(service, flow, harnessID); err != nil {
		writef(stderr, "hp tui: %v\n", err)
		return 1
	}
	return 0
}

func runAdd(service app.Service, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("add", stderr)
	label := fs.String("label", "", "harness label")
	links := stringSliceFlag{}
	fs.Var(&links, "link", "managed link to configure (repeat --link <id>=<path>; legacy --link <path> maps to root)")
	restart := fs.String("restart", "", "message to print after switching profiles")
	initial := fs.String("profile", "", "initial profile for existing roots")
	importSymlink := fs.Bool("import", false, "import external symlink target")
	registerSymlink := fs.Bool("register-external", false, "register external symlink as-is")
	if fs.Parse(args) != nil || fs.NArg() != 1 {
		if interactiveTerminal() {
			return runTUIFlow(service, tui.FlowAdd, "", stderr)
		}
		writeln(stderr, "usage: hp add <harness> --link <id>=<path> [--link <id>=<path> ...] [--label <label>] [--profile <name>] [--import|--register-external]")
		writeln(stderr, "       legacy single-link form: hp add <harness> --link <path> [--label <label>] [--profile <name>] [--import|--register-external]")
		return 1
	}
	harnessLinks, linkActions, err := parseConfigRootLinks(links.values, domain.Harness{}, true)
	if err != nil {
		writef(stderr, "hp add: %v\n", err)
		writeln(stderr, "usage: hp add <harness> --link <id>=<path> [--link <id>=<path> ...] [--label <label>] [--profile <name>] [--import|--register-external]")
		writeln(stderr, "       legacy single-link form: hp add <harness> --link <path> [--label <label>] [--profile <name>] [--import|--register-external]")
		return 1
	}
	if len(harnessLinks) == 0 {
		writef(stderr, "hp add: --link is required\n")
		writeln(stderr, "usage: hp add <harness> --link <id>=<path> [--link <id>=<path> ...] [--label <label>] [--profile <name>] [--import|--register-external]")
		writeln(stderr, "       legacy single-link form: hp add <harness> --link <path> [--label <label>] [--profile <name>] [--import|--register-external]")
		return 1
	}
	options := app.AddHarnessOptions{ID: fs.Arg(0), Label: *label, RestartHint: *restart, InitialProfile: *initial, ImportSymlink: *importSymlink, RegisterSymlink: *registerSymlink, LinkActions: linkActions}
	if len(harnessLinks) == 1 && isLegacySingleDirectoryLink(harnessLinks) {
		options.LinkPath = harnessLinks[0].Path
	} else {
		options.Links = harnessLinks
	}
	harness, err := service.AddHarness(options)
	if err != nil {
		writef(stderr, "hp add: %v\n", err)
		return 1
	}
	writef(stdout, "%s\t%s\t%s\n", harness.ID, harness.Label, formatHarnessLinksSummary(harness.LinksOrLegacy()))
	return 0
}

func runUpdate(service app.Service, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("update", stderr)
	label := fs.String("label", "", "harness label")
	restart := fs.String("restart", "", "message to print after switching profiles")
	removeOld := fs.Bool("remove-old", false, "remove old symlink after link update")
	links := stringSliceFlag{}
	fs.Var(&links, "link", "managed link to configure (repeat --link <id>=<path>; legacy --link <path> maps to root)")
	if fs.Parse(args) != nil || fs.NArg() != 1 {
		if interactiveTerminal() {
			return runTUIFlow(service, tui.FlowUpdate, "", stderr)
		}
		writeln(stderr, "usage: hp update <harness> [--label <label>] [--link <id>=<path> ...] [--restart <hint>] [--remove-old]")
		writeln(stderr, "       legacy single-link form: hp update <harness> [--label <label>] [--link <path>] [--restart <hint>] [--remove-old]")
		return 1
	}
	var updatedLinks []domain.HarnessLink
	var linkActions map[string]app.HarnessLinkAction
	if len(links.values) > 0 {
		var parseErr error
		allowLegacy := true
		harnessInfo, inspectErr := service.InspectHarness(fs.Arg(0))
		if inspectErr == nil {
			if len(harnessInfo.LinksOrLegacy()) > 1 {
				allowLegacy = false
			}
			updatedLinks, linkActions, parseErr = parseConfigRootLinks(links.values, harnessInfo, allowLegacy)
		} else {
			updatedLinks, linkActions, parseErr = parseConfigRootLinks(links.values, domain.Harness{}, allowLegacy)
		}
		if parseErr != nil {
			writef(stderr, "hp update: %v\n", parseErr)
			writeln(stderr, "usage: hp update <harness> [--label <label>] [--link <id>=<path> ...] [--restart <hint>] [--remove-old]")
			writeln(stderr, "       legacy single-link form: hp update <harness> [--label <label>] [--link <path>] [--restart <hint>] [--remove-old]")
			return 1
		}
	}
	var linkPath string
	if len(updatedLinks) == 1 && isLegacySingleDirectoryLink(updatedLinks) {
		linkPath = updatedLinks[0].Path
	}
	harness, err := service.UpdateHarness(app.UpdateHarnessOptions{ID: fs.Arg(0), Label: *label, LinkPath: linkPath, Links: updatedLinks, LinkActions: linkActions, RestartHint: *restart, RemoveOld: *removeOld})
	if err != nil {
		writef(stderr, "hp update: %v\n", err)
		return 1
	}
	writef(stdout, "%s\t%s\t%s\n", harness.ID, harness.Label, formatHarnessLinksSummary(harness.LinksOrLegacy()))
	return 0
}

func runDeleteHarness(service app.Service, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("delete", stderr)
	mode := fs.String("mode", "keep-root", "restore, keep-root, or delete-all")
	restore := fs.String("profile", "", "profile to restore")
	confirm := fs.String("confirm", "", "typed confirmation for delete-all")
	if fs.Parse(args) != nil || fs.NArg() != 1 {
		if interactiveTerminal() {
			return runTUIFlow(service, tui.FlowDeleteHarness, "", stderr)
		}
		writeln(stderr, "usage: hp delete <harness> [--mode keep-root|restore|delete-all] [--profile <name>] [--confirm <harness>]")
		return 1
	}
	if err := service.DeleteHarness(app.DeleteHarnessOptions{ID: fs.Arg(0), Mode: *mode, RestoreProfile: *restore, Confirm: *confirm}); err != nil {
		writef(stderr, "hp delete: %v\n", err)
		return 1
	}
	writef(stdout, "deleted %s\n", fs.Arg(0))
	return 0
}

func runHarnessCommand(service app.Service, args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		writef(stderr, "hp: unknown command %q\n", args[0])
		printHelp(stderr)
		return 1
	}
	harnessID, command := args[0], args[1]
	rest := args[2:]
	switch command {
	case "ls", "list":
		return runProfileList(service, harnessID, stdout, stderr)
	case "current", "cur":
		return runCurrentProfile(service, harnessID, stdout, stderr)
	case "switch", "sw":
		return runSwitchProfile(service, harnessID, rest, stdout, stderr)
	case "adopt":
		return runAdoptProfile(service, harnessID, rest, stdout, stderr)
	case "clone", "cp":
		return runCloneProfile(service, harnessID, rest, stdout, stderr)
	case "delete", "rm":
		return runDeleteProfile(service, harnessID, rest, stdout, stderr)
	case "where":
		return runHarnessWhere(service, harnessID, rest, stdout, stderr)
	}
	writef(stderr, "hp %s: unknown command %q\n", harnessID, command)
	return 1
}

func runProfileList(service app.Service, harnessID string, stdout, stderr io.Writer) int {
	profiles, err := service.ListProfiles(harnessID)
	if err != nil {
		writef(stderr, "hp %s ls: %v\n", harnessID, err)
		return 1
	}
	for _, profile := range profiles {
		mark := ""
		if profile.Active {
			mark = "*"
		}
		writef(stdout, "%s\t%s\n", profile.Name, mark)
	}
	return 0
}

func runCurrentProfile(service app.Service, harnessID string, stdout, stderr io.Writer) int {
	profile, err := service.CurrentProfile(harnessID)
	if err != nil {
		writef(stderr, "hp %s current: %v\n", harnessID, err)
		return 1
	}
	if profile.External {
		writef(stdout, "external\t%s\n", profile.Path)
	} else if profile.Name != "" {
		writef(stdout, "%s\n", profile.Name)
	}
	return 0
}

func runSwitchProfile(service app.Service, harnessID string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return missingProfileArg(service, tui.FlowSwitch, harnessID, stderr, "usage: hp <harness> switch <profile>")
	}
	harness, err := service.SwitchProfile(harnessID, args[0])
	if err != nil {
		writef(stderr, "hp %s switch: %v\n", harnessID, err)
		return 1
	}
	writef(stdout, "switched %s to %s\n", harnessID, args[0])
	if strings.TrimSpace(harness.RestartHint) != "" {
		writeln(stdout, harness.RestartHint)
	}
	return 0
}

func runAdoptProfile(service app.Service, harnessID string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return missingProfileArg(service, tui.FlowAdopt, harnessID, stderr, "usage: hp <harness> adopt <profile>")
	}
	if err := service.AdoptProfile(harnessID, args[0]); err != nil {
		writef(stderr, "hp %s adopt: %v\n", harnessID, err)
		return 1
	}
	writef(stdout, "adopted %s\n", args[0])
	return 0
}

func runCloneProfile(service app.Service, harnessID string, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("clone", stderr)
	materialize := fs.Bool("materialize", false, "materialize symlinks")
	if fs.Parse(args) != nil || fs.NArg() != 2 {
		if interactiveTerminal() {
			return runTUIFlow(service, tui.FlowClone, harnessID, stderr)
		}
		writeln(stderr, "usage: hp <harness> clone <source> <target> [--materialize]")
		return 1
	}
	if err := service.CloneProfile(harnessID, fs.Arg(0), fs.Arg(1), *materialize); err != nil {
		writef(stderr, "hp %s clone: %v\n", harnessID, err)
		return 1
	}
	writef(stdout, "cloned %s to %s\n", fs.Arg(0), fs.Arg(1))
	return 0
}

func runDeleteProfile(service app.Service, harnessID string, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("profile-delete", stderr)
	yes := fs.Bool("yes", false, "confirm profile deletion")
	if fs.Parse(args) != nil || fs.NArg() != 1 {
		if interactiveTerminal() {
			return runTUIFlow(service, tui.FlowDeleteProfile, harnessID, stderr)
		}
		writeln(stderr, "usage: hp <harness> delete <profile> --yes")
		return 1
	}
	if err := service.DeleteProfile(harnessID, fs.Arg(0), *yes); err != nil {
		writef(stderr, "hp %s delete: %v\n", harnessID, err)
		return 1
	}
	writef(stdout, "deleted %s\n", fs.Arg(0))
	return 0
}

func runHarnessWhere(service app.Service, harnessID string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		harness, err := service.InspectHarness(harnessID)
		if err != nil {
			writef(stderr, "hp %s where: %v\n", harnessID, err)
			return 1
		}
		links := harness.LinksOrLegacy()
		if isLegacySingleDirectoryLink(links) {
			writef(stdout, "link: %s\n", links[0].Path)
			return 0
		}
		for _, link := range links {
			writef(stdout, "link:\t%s\t%s\t%s\n", link.ID, link.Kind, link.Path)
		}
		return 0
	}
	if len(args) == 1 {
		path, err := service.ProfilePath(harnessID, args[0])
		if err != nil {
			writef(stderr, "hp %s where: %v\n", harnessID, err)
			return 1
		}
		writeln(stdout, path)
		return 0
	}
	writef(stderr, "usage: hp <harness> where [profile]\n")
	return 1
}

func missingProfileArg(service app.Service, flow tui.Flow, harnessID string, stderr io.Writer, usage string) int {
	if interactiveTerminal() {
		return runTUIFlow(service, flow, harnessID, stderr)
	}
	writeln(stderr, usage)
	return 1
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func runWhere(service app.Service, stdout, stderr io.Writer) int {
	paths, err := service.Where()
	if err != nil {
		writef(stderr, "hp where: %v\n", err)
		return 1
	}

	writef(stdout, "config: %s\n", paths.ConfigPath)
	writef(stdout, "harnesses: %s\n", paths.HarnessesRoot)
	return 0
}

func runList(service app.Service, stdout, stderr io.Writer) int {
	harnesses, err := service.ListHarnesses()
	if err != nil {
		writef(stderr, "hp ls: %v\n", err)
		return 1
	}

	sort.Slice(harnesses, func(i, j int) bool { return harnesses[i].ID < harnesses[j].ID })
	for _, harness := range harnesses {
		writef(stdout, "%s\t%s\t%s\n", harness.ID, harness.Label, formatHarnessLinksSummary(harness.LinksOrLegacy()))
	}
	return 0
}

func formatHarnessLinksSummary(links []domain.HarnessLink) string {
	if len(links) == 0 {
		return ""
	}
	if isLegacySingleDirectoryLink(links) {
		return links[0].Path
	}

	summaries := make([]string, 0, len(links))
	for _, link := range links {
		summaries = append(summaries, formatHarnessLink(link))
	}
	return strings.Join(summaries, ",")
}

type stringSliceFlag struct {
	values []string
}

func (f *stringSliceFlag) String() string {
	return strings.Join(f.values, ",")
}

func (f *stringSliceFlag) Set(value string) error {
	f.values = append(f.values, value)
	return nil
}

func parseConfigRootLinks(values []string, existing domain.Harness, legacyAllowed bool) ([]domain.HarnessLink, map[string]app.HarnessLinkAction, error) {
	if len(values) == 0 {
		return nil, nil, nil
	}

	hasLegacy := false
	for _, raw := range values {
		if strings.Contains(raw, "=") {
			continue
		}
		hasLegacy = true
	}
	if hasLegacy {
		if !legacyAllowed {
			return nil, nil, errors.New("legacy syntax is not supported with multiple --link values")
		}
		if len(values) != 1 {
			return nil, nil, errors.New("legacy --link <path> syntax supports only one --link value")
		}
	}

	existingLinks := existing.LinksOrLegacy()
	existingByID := map[string]domain.HarnessLink{}
	for _, existingLink := range existingLinks {
		existingByID[strings.ToLower(strings.TrimSpace(existingLink.ID))] = existingLink
	}

	parsedLinks := make([]domain.HarnessLink, 0, len(values))
	linkActions := map[string]app.HarnessLinkAction{}
	seenIDs := map[string]struct{}{}

	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil, nil, errors.New("link specification is required")
		}
		raw, action := splitLinkAction(raw)

		if strings.Contains(raw, "=") {
			parts := strings.SplitN(raw, "=", 2)
			id := strings.TrimSpace(parts[0])
			path := strings.TrimSpace(parts[1])
			if id == "" {
				return nil, nil, fmt.Errorf("link ID is required in %q", raw)
			}
			if path == "" {
				return nil, nil, fmt.Errorf("link path is required in %q", raw)
			}
			if err := domain.ValidateHarnessLinkID(id); err != nil {
				return nil, nil, fmt.Errorf("invalid link ID in %q: %w", raw, err)
			}
			var kind domain.HarnessLinkKind
			if existing, ok := existingByID[strings.ToLower(id)]; ok {
				kind = existing.Kind
			} else {
				inferredKind, err := inferLinkKindFromPath(path)
				if err != nil {
					return nil, nil, fmt.Errorf("invalid %s kind: %w", raw, err)
				}
				kind = inferredKind
			}
			idKey := strings.ToLower(id)
			if _, exists := seenIDs[idKey]; exists {
				return nil, nil, fmt.Errorf("duplicate link id %q", id)
			}
			seenIDs[idKey] = struct{}{}
			parsedLinks = append(parsedLinks, domain.HarnessLink{ID: id, Path: path, Kind: kind})
			if action != "" {
				linkActions[idKey] = action
			}
			continue
		}

		if hasLegacy {
			id := domain.LegacyDefaultLinkID
			if len(existingLinks) == 1 {
				id = strings.TrimSpace(existingLinks[0].ID)
			}
			path := raw
			var inferredKind domain.HarnessLinkKind
			if len(existingLinks) == 1 {
				inferredKind = existingLinks[0].Kind
			} else {
				kind, err := inferLinkKindFromPath(path)
				if err != nil {
					return nil, nil, fmt.Errorf("invalid %s kind: %w", raw, err)
				}
				inferredKind = kind
			}
			parsedLinks = append(parsedLinks, domain.HarnessLink{ID: id, Path: path, Kind: inferredKind})
			if action != "" {
				linkActions[strings.ToLower(strings.TrimSpace(id))] = action
			}
			continue
		}
	}

	if err := validateHarnessLinks(parsedLinks); err != nil {
		return nil, nil, err
	}
	return parsedLinks, linkActions, nil
}

func splitLinkAction(raw string) (string, app.HarnessLinkAction) {
	parts := strings.Split(raw, "|")
	if len(parts) < 2 {
		return raw, ""
	}
	action := parseLinkAction(parts[len(parts)-1])
	if action == "" {
		return raw, ""
	}
	return strings.Join(parts[:len(parts)-1], "|"), action
}

func parseLinkAction(raw string) app.HarnessLinkAction {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(app.HarnessLinkActionImport):
		return app.HarnessLinkActionImport
	case string(app.HarnessLinkActionRegister):
		return app.HarnessLinkActionRegister
	case string(app.HarnessLinkActionCreate):
		return app.HarnessLinkActionCreate
	default:
		return ""
	}
}

func inferLinkKindFromPath(path string) (domain.HarnessLinkKind, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.HarnessLinkKindDir, nil
		}
		return "", err
	}
	if info.IsDir() {
		return domain.HarnessLinkKindDir, nil
	}
	if info.Mode().IsRegular() {
		return domain.HarnessLinkKindFile, nil
	}
	return "", fmt.Errorf("%s is neither a directory nor regular file", path)
}

func validateHarnessLinks(links []domain.HarnessLink) error {
	if len(links) == 0 {
		return nil
	}
	for _, link := range links {
		if err := link.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func formatHarnessLink(link domain.HarnessLink) string {
	return fmt.Sprintf("%s:%s:%s", link.ID, link.Kind, link.Path)
}

func isLegacySingleDirectoryLink(links []domain.HarnessLink) bool {
	if len(links) != 1 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(links[0].ID), domain.LegacyDefaultLinkID) && string(links[0].Kind) == string(domain.HarnessLinkKindDir)
}

func write(w io.Writer, text string) {
	_, _ = fmt.Fprint(w, text)
}

func writef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func writeln(w io.Writer, args ...any) {
	_, _ = fmt.Fprintln(w, args...)
}
