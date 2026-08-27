package domain

import "testing"

func TestTargetValidation(t *testing.T) {
	valid := ProtectedTarget{TargetType: "building", Name: "竖井", DistanceMeters: 10, AllowedPpvMmPerSec: 8, BaselinePpvMmPerSec: 0.2, MeasurementNote: "采样"}
	if err := ValidateTarget(valid); err != nil {
		t.Fatal(err)
	}
	valid.BaselinePpvMmPerSec = 9
	if ErrorCodeOf(ValidateTarget(valid)) != CodeValidation {
		t.Fatal("未拒绝超过允许值的基线")
	}
}

func TestMonitoringRequiresKnownTarget(t *testing.T) {
	p := MonitoringPlan{SensorPoints: []SensorPoint{{Name: "S1", TargetID: "unknown", Location: "基础"}}, SampleRateHz: 2000, TriggerPpvMmPerSec: 5, EvacuationRule: "撤离", RemainingRisk: "可控"}
	if err := ValidateMonitoring(p, []ProtectedTarget{{TargetID: "known"}}); err == nil {
		t.Fatal("未拒绝未知保护对象测点")
	}
}

func TestTargetBatchDetectsNormalizedDuplicatesAndKeepsDifferentTypes(t *testing.T) {
	existing := []ProtectedTarget{{TargetID: "target-existing", TargetType: "building", Name: "办公楼"}}
	batch := []ProtectedTarget{
		{TargetType: "building", Name: "  办公楼  ", DistanceMeters: 10, AllowedPpvMmPerSec: 8, BaselinePpvMmPerSec: 1, MeasurementNote: "测量"},
		{TargetType: "pipeline", Name: "办公楼", DistanceMeters: 10, AllowedPpvMmPerSec: 8, BaselinePpvMmPerSec: 1, MeasurementNote: "测量"},
		{TargetType: "pipeline", Name: " 办公楼 ", DistanceMeters: 12, AllowedPpvMmPerSec: 9, BaselinePpvMmPerSec: 1, MeasurementNote: "测量"},
	}
	issues := ValidateTargetBatch(batch, existing)
	if len(issues) != 2 || issues[0].ExistingTargetID != "target-existing" || issues[1].ConflictingItemIndex != 2 {
		t.Fatalf("批次重复结果错误: %#v", issues)
	}
}

func TestBaselineAndMonitoringPrechecks(t *testing.T) {
	targets := []ProtectedTarget{
		{TargetID: "building", TargetType: "building", Name: "楼", DistanceMeters: 10, AllowedPpvMmPerSec: 10, BaselinePpvMmPerSec: 9, MeasurementNote: "测量"},
		{TargetID: "pipeline", TargetType: "pipeline", Name: "管", DistanceMeters: 20, AllowedPpvMmPerSec: 8, BaselinePpvMmPerSec: 1, MeasurementNote: "测量"},
	}
	precheck := PrecheckBaseline("case-one", targets)
	if precheck.Ready || len(precheck.MissingTypes) != 1 || precheck.MissingTypes[0] != "equipment" || precheck.ControlTargetID != "building" {
		t.Fatalf("基线预检结果错误: %#v", precheck)
	}
	targets = append(targets, ProtectedTarget{TargetID: "equipment", TargetType: "equipment", Name: "设备", DistanceMeters: 30, AllowedPpvMmPerSec: 6, BaselinePpvMmPerSec: 1, MeasurementNote: "测量"})
	assessment := AssessmentSnapshot{ControlTargetID: "pipeline", PredictedTargets: []PredictedTarget{{TargetID: "pipeline", PredictedPpvMmPerSec: 7, MarginMmPerSec: 1}}}
	plan := MonitoringPlan{SensorPoints: []SensorPoint{{Name: "S1", TargetID: "building", Location: "L1"}}, TriggerPpvMmPerSec: 11}
	validation := BuildMonitoringValidation(plan, targets, assessment)
	if validation.Ready || len(validation.UncoveredTargetIDs) != 1 || validation.UncoveredTargetIDs[0] != "pipeline" || len(validation.ThresholdIssues) != 1 {
		t.Fatalf("监测联锁结果错误: %#v", validation)
	}
}
