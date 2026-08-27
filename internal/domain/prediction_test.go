package domain

import (
	"testing"
	"time"
)

func TestAssessCreatesStableBlockingFinding(t *testing.T) {
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	r := DesignRevision{RevisionID: "rev-one", MaxChargePerDelayKg: 100, DelaySequenceMs: []int{0, 25}, PropagationK: 100, PropagationAlpha: 1.6}
	targets := []ProtectedTarget{{TargetID: "target-one", DistanceMeters: 20, AllowedPpvMmPerSec: 10}}
	first := Assess(r, targets, now, "assessment-one")
	second := Assess(r, targets, now, "assessment-two")
	if !HasBlockers(first) || len(first.Findings) != 1 {
		t.Fatalf("期望一个超限阻断项，得到 %#v", first.Findings)
	}
	if first.Findings[0].FindingID != second.Findings[0].FindingID {
		t.Fatal("相同输入没有产生稳定 findingId")
	}
	if first.PredictedTargets[0].Pass {
		t.Fatal("超限预测被标记为通过")
	}
}

func TestDelayAnomaly(t *testing.T) {
	for _, sequence := range [][]int{{0, 0}, {25, 10}, {0, 4}} {
		if !DelayAnomaly(sequence) {
			t.Fatalf("未识别异常延期序列 %v", sequence)
		}
	}
	if DelayAnomaly([]int{0, 5, 20}) {
		t.Fatal("有效延期序列被误判")
	}
}

func TestMissingPropagationParametersBecomeFinding(t *testing.T) {
	r := DesignRevision{RevisionID: "rev-missing", MaxChargePerDelayKg: 1, DelaySequenceMs: []int{0, 25}}
	a := Assess(r, []ProtectedTarget{{TargetID: "target-one", DistanceMeters: 10, AllowedPpvMmPerSec: 8}}, time.Now(), "assessment-missing")
	if !HasBlockers(a) || a.Findings[0].Code != "PROPAGATION_PARAMETER_MISSING" {
		t.Fatalf("缺失传播参数未形成阻断项: %#v", a.Findings)
	}
	if len(a.AllowedCharges) != 1 || a.AllowedCharges[0].Status != "unavailable" || a.AllowedCharges[0].MaxAllowedChargeKg != nil {
		t.Fatalf("缺失传播参数时反算状态错误: %#v", a.AllowedCharges)
	}
}

func TestAssessBackCalculatesUniqueControlTarget(t *testing.T) {
	r := DesignRevision{RevisionID: "rev-control", MaxChargePerDelayKg: 4, DelaySequenceMs: []int{0, 25}, PropagationK: 100, PropagationAlpha: 2}
	targets := []ProtectedTarget{{TargetID: "near", DistanceMeters: 20, AllowedPpvMmPerSec: 1}, {TargetID: "far", DistanceMeters: 40, AllowedPpvMmPerSec: 20}}
	a := Assess(r, targets, time.Now(), "assessment-control")
	if a.ControlTargetID != "near" || a.ControlChargeKg == nil || *a.ControlChargeKg != 4 {
		t.Fatalf("控制目标或控制药量错误: %#v", a)
	}
	if a.ChargeMarginKg == nil || *a.ChargeMarginKg != 0 || a.ChargeMarginPercent == nil || *a.ChargeMarginPercent != 0 {
		t.Fatalf("当前药量余量错误: %#v", a)
	}
	controlCount := 0
	for _, limit := range a.AllowedCharges {
		if limit.Control {
			controlCount++
		}
	}
	if controlCount != 1 {
		t.Fatalf("控制目标数量=%d", controlCount)
	}
}

func TestEvidenceDigestChangesWithInputs(t *testing.T) {
	f := CaseFile{Case: BlastCase{CaseID: "case-one"}, Targets: []ProtectedTarget{{TargetID: "a", DistanceMeters: 20}}}
	one := EvidenceDigest(f)
	f.Targets[0].DistanceMeters = 21
	if one == EvidenceDigest(f) {
		t.Fatal("证据输入变化后摘要未变化")
	}
}
