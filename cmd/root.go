package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cliresource "k8s.io/cli-runtime/pkg/resource"

	"github.com/itzik-elayev/kubectl-unitconv/pkg/convert"
	"github.com/itzik-elayev/kubectl-unitconv/pkg/k8sfield"
)

type rootOptions struct {
	family      string
	precision   int
	from        string
	container   string
	requirement string
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

	configFlags.AddFlags(cmd.Flags())

	return cmd
}

func runRoot(out io.Writer, args []string, opts *rootOptions, configFlags *genericclioptions.ConfigFlags) error {
	if err := validateArgs(args, opts); err != nil {
		return err
	}

	if opts.from != "" {
		return runFrom(out, args, opts, configFlags)
	}
	return runLiteral(out, args, opts)
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

func runLiteral(out io.Writer, args []string, opts *rootOptions) error {
	rawInput := args[0]

	q, err := resource.ParseQuantity(rawInput)
	if err != nil {
		return fmt.Errorf("invalid quantity %q: %w", rawInput, err)
	}

	family := convert.DetectFamily(rawInput, "", opts.family)

	var targetUnit string
	if len(args) == 2 {
		targetUnit = args[1]
	}

	return printQuantity(out, "", q, targetUnit, family, opts.precision)
}

func runFrom(out io.Writer, args []string, opts *rootOptions, configFlags *genericclioptions.ConfigFlags) error {
	fromSpec, err := k8sfield.ParseFromSpec(opts.from)
	if err != nil {
		return err
	}

	var targetUnit string
	if len(args) == 1 {
		targetUnit = args[0]
	}

	obj, err := fetchResource(configFlags, fromSpec.ResourceArg)
	if err != nil {
		return err
	}

	if fromSpec.FieldPath != nil {
		return runFromExplicitPath(out, obj, fromSpec.FieldPath, targetUnit, opts)
	}
	return runFromDefaultPath(out, obj, targetUnit, opts)
}

func fetchResource(configFlags *genericclioptions.ConfigFlags, resourceArg string) (*unstructured.Unstructured, error) {
	namespace, _, err := configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, fmt.Errorf("resolving namespace: %w", err)
	}

	result := cliresource.NewBuilder(configFlags).
		Unstructured().
		NamespaceParam(namespace).DefaultNamespace().
		ResourceTypeOrNameArgs(false, resourceArg).
		Flatten().
		Do()

	runtimeObj, err := result.Object()
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", resourceArg, err)
	}

	obj, ok := runtimeObj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("unexpected object type for %s", resourceArg)
	}
	return obj, nil
}

func runFromExplicitPath(out io.Writer, obj *unstructured.Unstructured, path []k8sfield.PathSegment, targetUnit string, opts *rootOptions) error {
	rawValue, err := k8sfield.WalkFieldPath(obj.Object, path)
	if err != nil {
		return err
	}

	q, err := resource.ParseQuantity(rawValue)
	if err != nil {
		return fmt.Errorf("parsing quantity %q from %s: %w", rawValue, obj.GetName(), err)
	}

	fieldHint := path[len(path)-1].Field
	family := convert.DetectFamily(rawValue, fieldHint, opts.family)

	return printQuantity(out, "", q, targetUnit, family, opts.precision)
}

func runFromDefaultPath(out io.Writer, obj *unstructured.Unstructured, targetUnit string, opts *rootOptions) error {
	family := convert.DetectFamily("", "", opts.family)

	fromOpts := k8sfield.FromOptions{Container: opts.container, Requirement: opts.requirement}
	readings, err := k8sfield.ResolveDefault(obj, fromOpts, family)
	if err != nil {
		return err
	}

	for _, reading := range readings {
		q, err := resource.ParseQuantity(reading.RawValue)
		if err != nil {
			return fmt.Errorf("parsing quantity %q from %s: %w", reading.RawValue, obj.GetName(), err)
		}

		if err := printQuantity(out, readingLabel(reading), q, targetUnit, family, opts.precision); err != nil {
			return err
		}
	}
	return nil
}

func readingLabel(r k8sfield.Reading) string {
	if r.Container == "" {
		return ""
	}

	kind := "container"
	if r.IsInit {
		kind = "init container"
	}
	if r.Requirement == "" {
		return fmt.Sprintf("%s %s", kind, r.Container)
	}
	return fmt.Sprintf("%s %s (%s)", kind, r.Container, r.Requirement)
}

// printQuantity renders a single quantity: converted to targetUnit if given,
// or every unit in its family otherwise. label prefixes the output to
// distinguish readings fanned out across multiple containers/requirements.
func printQuantity(out io.Writer, label string, q resource.Quantity, targetUnit string, family convert.Family, precision int) error {
	if targetUnit != "" {
		converted, err := convert.ToUnit(q, targetUnit, precision, family)
		if err != nil {
			return err
		}
		if label == "" {
			fmt.Fprintln(out, converted)
			return nil
		}
		fmt.Fprintf(out, "%s: %s\n", label, converted)
		return nil
	}

	indent := ""
	if label != "" {
		fmt.Fprintf(out, "%s:\n", label)
		indent = "  "
	}
	for _, result := range convert.ShowAll(q, family, precision) {
		fmt.Fprintf(out, "%s%-20s %s\n", indent, result.Unit.Label, result.Value)
	}
	return nil
}
