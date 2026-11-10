package ensure

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Apply creates or updates obj using CreateOrUpdate. When dryRun is true, no API writes occur.
func Apply(ctx context.Context, c client.Client, obj client.Object, owner client.Object, dryRun bool, mutate func() error) (controllerutil.OperationResult, error) {
	if dryRun {
		return controllerutil.OperationResultNone, nil
	}
	result, err := controllerutil.CreateOrUpdate(ctx, c, obj, func() error {
		if owner != nil {
			if err := controllerutil.SetControllerReference(owner, obj, c.Scheme()); err != nil {
				return err
			}
		}
		if mutate != nil {
			return mutate()
		}
		return nil
	})
	return result, err
}
