package terraform

import (
	"github.com/hashicorp/terraform-exec/tfexec"
)

func (d *DriverConfig) GetInitOptions() []tfexec.InitOption {
	var initOptions []tfexec.InitOption

	if !d.Backend {
		initOptions = append(initOptions, tfexec.Backend(d.Backend))
	}

	for i := range d.BackendConfig {
		initOptions = append(initOptions, tfexec.BackendConfig(d.BackendConfig[i]))
	}

	if d.ForceCopy {
		initOptions = append(initOptions, tfexec.ForceCopy(d.ForceCopy))
	}

	if d.FromModule != "" {
		initOptions = append(initOptions, tfexec.FromModule(d.FromModule))
	}

	if !d.Get {
		initOptions = append(initOptions, tfexec.Get(d.Get))
	}

	if d.Reconfigure {
		initOptions = append(initOptions, tfexec.Reconfigure(d.Reconfigure))
	}

	if d.Upgrade {
		initOptions = append(initOptions, tfexec.Upgrade(d.Upgrade))
	}

	return initOptions
}

func (d *DriverConfig) GetPlanOptions() []tfexec.PlanOption {
	var planOptions []tfexec.PlanOption

	for i := range d.Target {
		planOptions = append(planOptions, tfexec.Target(d.Target[i]))
	}

	for i := range d.Var {
		planOptions = append(planOptions, tfexec.Var(d.Var[i]))
	}

	for i := range d.VarFile {
		planOptions = append(planOptions, tfexec.VarFile(d.VarFile[i]))
	}

	if d.PlanFile != "" {
		planOptions = append(planOptions, tfexec.Out(d.PlanFile))
	}

	if !d.Refresh {
		planOptions = append(planOptions, tfexec.Refresh(d.Refresh))
	}

	if d.Destroy {
		planOptions = append(planOptions, tfexec.Destroy(d.Destroy))
	}

	if !d.Lock {
		planOptions = append(planOptions, tfexec.Lock(d.Lock))
	}

	if d.LockTimeout != defaultLockTimeout {
		planOptions = append(planOptions, tfexec.LockTimeout(d.LockTimeout))
	}

	if d.Parallelism != defaultParallelism {
		planOptions = append(planOptions, tfexec.Parallelism(d.Parallelism))
	}

	for i := range d.Replace {
		planOptions = append(planOptions, tfexec.Replace(d.Replace[i]))
	}
	return planOptions
}

func (d *DriverConfig) GetForceUnlockOptions() []tfexec.ForceUnlockOption {
	var forceUnlockOptions []tfexec.ForceUnlockOption
	return forceUnlockOptions
}

func (d *DriverConfig) GetApplyOptions() []tfexec.ApplyOption {
	var applyOptions []tfexec.ApplyOption

	for i := range d.Target {
		applyOptions = append(applyOptions, tfexec.Target(d.Target[i]))
	}

	for i := range d.Var {
		applyOptions = append(applyOptions, tfexec.Var(d.Var[i]))
	}

	for i := range d.VarFile {
		applyOptions = append(applyOptions, tfexec.VarFile(d.VarFile[i]))
	}

	if d.PlanFile != "" {
		applyOptions = append(applyOptions, tfexec.DirOrPlan(d.PlanFile))
	}

	if !d.Refresh {
		applyOptions = append(applyOptions, tfexec.Refresh(d.Refresh))
	}

	if d.Backup != "" {
		applyOptions = append(applyOptions, tfexec.Backup(d.Backup))
	}

	if !d.Lock {
		applyOptions = append(applyOptions, tfexec.Lock(d.Lock))
	}

	if d.LockTimeout != defaultLockTimeout {
		applyOptions = append(applyOptions, tfexec.LockTimeout(d.LockTimeout))
	}

	if d.Parallelism != defaultParallelism {
		applyOptions = append(applyOptions, tfexec.Parallelism(d.Parallelism))
	}

	for i := range d.Replace {
		applyOptions = append(applyOptions, tfexec.Replace(d.Replace[i]))
	}

	if d.StateOut != "" {
		applyOptions = append(applyOptions, tfexec.StateOut(d.StateOut))
	}

	return applyOptions
}

func (d *DriverConfig) GetDestroyOptions() []tfexec.DestroyOption {
	var destroyOptions []tfexec.DestroyOption

	for i := range d.Target {
		destroyOptions = append(destroyOptions, tfexec.Target(d.Target[i]))
	}

	for i := range d.Var {
		destroyOptions = append(destroyOptions, tfexec.Var(d.Var[i]))
	}

	for i := range d.VarFile {
		destroyOptions = append(destroyOptions, tfexec.VarFile(d.VarFile[i]))
	}

	if !d.Refresh {
		destroyOptions = append(destroyOptions, tfexec.Refresh(d.Refresh))
	}

	if d.Backup != "" {
		destroyOptions = append(destroyOptions, tfexec.Backup(d.Backup))
	}

	if !d.Lock {
		destroyOptions = append(destroyOptions, tfexec.Lock(d.Lock))
	}

	if d.LockTimeout != defaultLockTimeout {
		destroyOptions = append(destroyOptions, tfexec.LockTimeout(d.LockTimeout))
	}

	if d.Parallelism != defaultParallelism {
		destroyOptions = append(destroyOptions, tfexec.Parallelism(d.Parallelism))
	}

	if d.StateOut != "" {
		destroyOptions = append(destroyOptions, tfexec.StateOut(d.StateOut))
	}

	return destroyOptions
}
