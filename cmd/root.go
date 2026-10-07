package cmd

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cliresource "k8s.io/cli-runtime/pkg/resource"

	"github.com/itzik-elayev/kubectl-unitconv/pkg/convert"
	"github.com/itzik-elayev/kubectl-unitconv/pkg/k8sfield"
	"github.com/itzik-elayev/kubectl-unitconv/pkg/render"
	"github.com/itzik-elayev/kubectl-unitconv/pkg/result"
)

type rootOptions struct {
	family      string
	precision   int
	from        string
	container   string
	requirement string
	output      string
	color       string
}

// NewRootCmd builds the kubectl-unitconv root command: convert a literal
// quantity, or one read live off a cluster resource via --from, to a target
// unit — or show every unit in its family when no target unit is given.
func NewRootCmd() *cobra.Command {
	opts := &rootOptions{}
	configFlags := genericclioptions.NewConfigFlags(true)

	cmd := &cobra.Command{
		Use:           "kubectl-unitconv [VALUE] [TARGET_UNIT]",
		Short:         "Convert Kubernetes resource.Quantity values between units",
		Args:          cobra.MaximumNArgs(2),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoot(cmd.OutOrStdout(), args, opts, configFlags)
		},
	}

	cmd.Flags().StringVar(&opts.family, "family", "auto", "force unit family: auto, cpu, or memory")
	cmd.Flags().IntVar(&opts.precision, "precision", 6, "decimal places in output")
	cmd.Flags().StringVar(&opts.from, "from", "", "read the quantity from a live resource: TYPE/NAME or TYPE/NAME:field.path")
	cmd.Flags().StringVar(&opts.container, "container", "", "with --from TYPE/NAME, limit to one container (default: all, incl. init)")
	cmd.Flags().StringVar(&opts.requirement, "requirement", "", "with --from TYPE/NAME, limit to requests or limits (default: both)")
	cmd.Flags().StringVar(&opts.output, "output", "auto", "output format: auto, table, plain, or json")
	cmd.Flags().StringVar(&opts.color, "color", "auto", "color mode: auto, always, or never")

	configFlags.AddFlags(cmd.Flags())

	return cmd
}

func runRoot(out io.Writer, args []string, opts *rootOptions, configFlags *genericclioptions.ConfigFlags) error {
	outputMode, err := render.ParseOutputMode(opts.output)
	if err != nil {
		return err
	}
	colorMode, err := render.ParseColorMode(opts.color)
	if err != nil {
		return err
	}
	if err := convert.ValidatePrecision(opts.precision); err != nil {
		return err
	}
	if err := validateRequirement(opts.requirement); err != nil {
		return err
	}
	forcedFamily, hasForcedFamily, err := convert.ParseFamilyFlag(opts.family)
	if err != nil {
		return err
	}

	var fromSpec k8sfield.FromSpec
	if opts.from != "" {
		fromSpec, err = k8sfield.ParseFromSpec(opts.from)
		if err != nil {
			return err
		}
	}
	if err := validateFlagApplicability(opts, fromSpec); err != nil {
		return err
	}
	if err := validateArgs(args, opts); err != nil {
		return err
	}

	var results []result.Result
	if opts.from != "" {
		results, err = buildFromResults(fromSpec, args, opts, forcedFamily, hasForcedFamily, configFlags)
	} else {
		results, err = buildLiteralResults(args, opts, forcedFamily, hasForcedFamily)
	}
	if err != nil {
		return err
	}

	renderResults(out, results, outputMode, colorMode)
	return nil
}

func validateArgs(args []string, opts *rootOptions) error {
	if opts.from != "" {
		if len(args) > 1 {
			return fmt.Errorf("at most one TARGET_UNIT argument is allowed together with --from")
		}
		return nil
	}

	if len(args) < 1 {
		return fmt.Errorf("VALUE argument is required (or use --from)")
	}
	return nil
}

// validateFlagApplicability rejects flags that don't apply to the selected
// input mode, rather than silently ignoring them.
func validateFlagApplicability(opts *rootOptions, fromSpec k8sfield.FromSpec) error {
	if opts.container == "" && opts.requirement == "" {
		return nil
	}
	if opts.from == "" {
		return fmt.Errorf("--container and --requirement require --from")
	}
	if fromSpec.FieldPath != nil {
		return fmt.Errorf("--container and --requirement do not apply to an explicit --from field path")
	}
	return nil
}

func validateRequirement(value string) error {
	switch value {
	case "", "requests", "limits":
		return nil
	default:
		return fmt.Errorf("invalid --requirement value %q: must be requests or limits", value)
	}
}

func renderResults(w io.Writer, results []result.Result, outputMode render.OutputMode, colorMode render.ColorMode) {
	isTerminal := isTerminalWriter(w)

	switch render.ResolveOutputMode(outputMode, isTerminal) {
	case render.OutputJSON:
		data, err := render.JSON(results)
		if err != nil {
			fmt.Fprintln(w, "{}") // JSON output must never fail to be valid JSON
			return
		}
		fmt.Fprintln(w, data)
	case render.OutputTable:
		colorEnabled := render.ResolveColorEnabled(colorMode, isTerminal, os.Getenv("NO_COLOR"))
		render.Table(w, results, colorEnabled, terminalWidth(w, isTerminal))
	default:
		render.Plain(w, results)
	}
}

func buildLiteralResults(args []string, opts *rootOptions, forcedFamily convert.Family, hasForcedFamily bool) ([]result.Result, error) {
	rawInput := args[0]
	var targetUnit string
	if len(args) == 2 {
		targetUnit = args[1]
	}

	q, err := resource.ParseQuantity(rawInput)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity %q: %w", rawInput, err)
	}

	family, err := convert.ResolveFamily(rawInput, "", targetUnit, forcedFamily, hasForcedFamily)
	if err != nil {
		return nil, err
	}

	base := result.Result{
		ResourceType: result.ResourceTypeFor(family, ""),
		Original:     rawInput,
	}

	return convertOne(base, q, family, targetUnit, opts.precision)
}

func buildFromResults(fromSpec k8sfield.FromSpec, args []string, opts *rootOptions, forcedFamily convert.Family, hasForcedFamily bool, configFlags *genericclioptions.ConfigFlags) ([]result.Result, error) {
	var targetUnit string
	if len(args) == 1 {
		targetUnit = args[0]
	}

	obj, err := fetchResource(configFlags, fromSpec.ResourceArg)
	if err != nil {
		return nil, err
	}

	base := result.Result{
		Kind:      obj.GetKind(),
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	if fromSpec.FieldPath != nil {
		return buildFromExplicitPath(obj, base, fromSpec.FieldPath, targetUnit, opts, forcedFamily, hasForcedFamily)
	}
	return buildFromDefaultPath(obj, base, targetUnit, opts, forcedFamily, hasForcedFamily)
}

func fetchResource(configFlags *genericclioptions.ConfigFlags, resourceArg string) (*unstructured.Unstructured, error) {
	namespace, _, err := configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, fmt.Errorf("resolving namespace: %w", err)
	}

	queryResult := cliresource.NewBuilder(configFlags).
		Unstructured().
		NamespaceParam(namespace).DefaultNamespace().
		ResourceTypeOrNameArgs(false, resourceArg).
		Flatten().
		Do()

	runtimeObj, err := queryResult.Object()
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", resourceArg, err)
	}

	obj, ok := runtimeObj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("unexpected object type for %s", resourceArg)
	}
	return obj, nil
}

func buildFromExplicitPath(obj *unstructured.Unstructured, base result.Result, path []k8sfield.PathSegment, targetUnit string, opts *rootOptions, forcedFamily convert.Family, hasForcedFamily bool) ([]result.Result, error) {
	rawValue, err := k8sfield.WalkFieldPath(obj.Object, path)
	if err != nil {
		return nil, err
	}

	q, err := resource.ParseQuantity(rawValue)
	if err != nil {
		return nil, fmt.Errorf("parsing quantity %q from %s: %w", rawValue, obj.GetName(), err)
	}

	fieldHint := path[len(path)-1].Field
	family, err := convert.ResolveFamily(rawValue, fieldHint, targetUnit, forcedFamily, hasForcedFamily)
	if err != nil {
		return nil, err
	}

	base.ResourceType = result.ResourceTypeFor(family, fieldHint)
	base.Original = rawValue

	return convertOne(base, q, family, targetUnit, opts.precision)
}

func buildFromDefaultPath(obj *unstructured.Unstructured, base result.Result, targetUnit string, opts *rootOptions, forcedFamily convert.Family, hasForcedFamily bool) ([]result.Result, error) {
	family, err := resolveDefaultPathFamily(targetUnit, forcedFamily, hasForcedFamily)
	if err != nil {
		return nil, err
	}

	resourceType := result.ResourceTypeFor(family, "")
	if isPersistentVolumeClaim(obj) {
		resourceType = result.ResourceTypeStorage
	}

	fromOpts := k8sfield.FromOptions{Container: opts.container, Requirement: opts.requirement}
	readings, err := k8sfield.ResolveDefault(obj, fromOpts, family)
	if err != nil {
		return nil, err
	}

	var results []result.Result
	for _, reading := range readings {
		q, err := resource.ParseQuantity(reading.RawValue)
		if err != nil {
			return nil, fmt.Errorf("parsing quantity %q from %s: %w", reading.RawValue, obj.GetName(), err)
		}

		readingBase := base
		readingBase.Container = reading.Container
		readingBase.IsInitContainer = reading.IsInit
		readingBase.ResourceType = resourceType
		readingBase.Requirement = reading.Requirement
		readingBase.Original = reading.RawValue

		converted, err := convertOne(readingBase, q, family, targetUnit, opts.precision)
		if err != nil {
			return nil, err
		}
		results = append(results, converted...)
	}

	return results, nil
}

func isPersistentVolumeClaim(obj *unstructured.Unstructured) bool {
	return obj.GetKind() == "PersistentVolumeClaim"
}

// resolveDefaultPathFamily decides which family a --from default-path lookup
// reads: an explicit --family wins; otherwise an unambiguous target unit
// (e.g. converting straight to "m") implies cpu without needing --family
// cpu too; otherwise memory, the documented ambiguous default.
func resolveDefaultPathFamily(targetUnit string, forced convert.Family, hasForced bool) (convert.Family, error) {
	if targetUnit != "" {
		if implied, ok := convert.FamilyFromTargetUnit(targetUnit); ok {
			if hasForced && implied != forced {
				return 0, fmt.Errorf("target unit %q implies %s, which conflicts with --family %s", targetUnit, implied, forced)
			}
			return implied, nil
		}
	}
	if hasForced {
		return forced, nil
	}
	return convert.FamilyMemory, nil
}

// convertOne converts a single quantity to targetUnit, or to every unit in
// its family when targetUnit is empty (show-all mode), filling in base's
// TargetUnit/Converted for each resulting Result.
func convertOne(base result.Result, q resource.Quantity, family convert.Family, targetUnit string, precision int) ([]result.Result, error) {
	if targetUnit == "" {
		// convert.ShowAll returns smallest-to-largest; print largest first,
		// since that's the more natural reading order for a unit listing.
		convResults := dropUnitsBelowOne(convert.ShowAll(q, family, precision))
		results := make([]result.Result, 0, len(convResults))
		for _, cr := range slices.Backward(convResults) {
			r := base
			r.TargetUnit = cr.Unit.Suffix
			r.Converted = cr.Value
			results = append(results, r)
		}
		return results, nil
	}

	unit, err := convert.ResolveTarget(family, targetUnit)
	if err != nil {
		return nil, err
	}
	converted, err := convert.ToUnit(q, targetUnit, precision, family)
	if err != nil {
		return nil, err
	}

	base.TargetUnit = unit.Suffix
	base.Converted = converted
	return []result.Result{base}, nil
}

// dropUnitsBelowOne removes show-all units where the quantity is less than
// one whole unit (e.g. 0.22Ti) — the unit is too coarse to be useful. If
// every unit is below one (a zero or sub-base-unit quantity), the finest
// unit alone is kept so there's always an answer. results must be ascending.
func dropUnitsBelowOne(results []convert.Result) []convert.Result {
	kept := make([]convert.Result, 0, len(results))
	for _, r := range results {
		if !r.IsBelowOne {
			kept = append(kept, r)
		}
	}
	if len(kept) == 0 && len(results) > 0 {
		return results[:1]
	}
	return kept
}
