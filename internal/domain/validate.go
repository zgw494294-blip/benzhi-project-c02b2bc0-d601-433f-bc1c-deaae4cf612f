package domain

import (
	"fmt"
	"math"
	"strings"
)

var targetTypes = map[string]bool{"building": true, "pipeline": true, "equipment": true}

func ValidateNewCase(site, zone string) error {
	if strings.TrimSpace(site) == "" || strings.TrimSpace(zone) == "" {
		return NewError(CodeValidation, "siteName 和 workZone 均为必填")
	}
	if len(site) > 120 || len(zone) > 120 {
		return NewError(CodeValidation, "名称长度不得超过 120")
	}
	return nil
}
func ValidateTarget(t ProtectedTarget) error {
	if !targetTypes[t.TargetType] {
		return NewError(CodeValidation, "targetType 必须为 building、pipeline 或 equipment")
	}
	if strings.TrimSpace(t.Name) == "" {
		return NewError(CodeValidation, "保护对象名称不能为空")
	}
	if !finite(t.DistanceMeters) || !finite(t.AllowedPpvMmPerSec) || !finite(t.BaselinePpvMmPerSec) {
		return NewError(CodeValidation, "距离和振速必须为有限数值")
	}
	if t.DistanceMeters <= 0 || t.AllowedPpvMmPerSec <= 0 {
		return NewError(CodeValidation, "距离和允许振速必须大于 0")
	}
	if t.BaselinePpvMmPerSec < 0 || t.BaselinePpvMmPerSec >= t.AllowedPpvMmPerSec {
		return NewError(CodeValidation, "基线振速必须非负且小于允许振速")
	}
	if strings.TrimSpace(t.MeasurementNote) == "" {
		return NewError(CodeValidation, "必须提供基线测量说明")
	}
	return nil
}
func ValidateRevision(r DesignRevision) error {
	if strings.TrimSpace(r.HolePattern) == "" || strings.TrimSpace(r.InitiationDirection) == "" {
		return NewError(CodeValidation, "孔网和起爆方向不能为空")
	}
	if !finite(r.MaxChargePerDelayKg) || !finite(r.PropagationK) || !finite(r.PropagationAlpha) {
		return NewError(CodeValidation, "药量和传播参数必须为有限数值")
	}
	if r.MaxChargePerDelayKg <= 0 {
		return NewError(CodeValidation, "单段最大药量必须大于 0")
	}
	if len(r.DelaySequenceMs) < 2 {
		return NewError(CodeValidation, "延期序列至少包含两个时刻")
	}
	if r.PropagationK < 0 || r.PropagationAlpha < 0 {
		return NewError(CodeValidation, "场地传播参数不能为负数")
	}
	return nil
}
func ValidateMonitoring(p MonitoringPlan, targets []ProtectedTarget) error {
	if len(p.SensorPoints) == 0 {
		return NewError(CodeValidation, "至少设置一个传感器点")
	}
	if p.SampleRateHz < 1000 {
		return NewError(CodeValidation, "采样频率不得低于 1000 Hz")
	}
	if !finite(p.TriggerPpvMmPerSec) || p.TriggerPpvMmPerSec <= 0 {
		return NewError(CodeValidation, "触发阈值必须大于 0")
	}
	if strings.TrimSpace(p.EvacuationRule) == "" || strings.TrimSpace(p.RemainingRisk) == "" {
		return NewError(CodeValidation, "撤离条件和剩余风险不能为空")
	}
	known := map[string]bool{}
	for _, t := range targets {
		known[t.TargetID] = true
	}
	names, locations := map[string]bool{}, map[string]bool{}
	for _, s := range p.SensorPoints {
		name, location := strings.TrimSpace(s.Name), strings.TrimSpace(s.Location)
		if name == "" || location == "" || !known[s.TargetID] {
			return NewError(CodeValidation, fmt.Sprintf("传感器点 %q 未关联有效保护对象", s.Name))
		}
		nameKey, locationKey := strings.ToLower(name), strings.ToLower(location)
		if names[nameKey] {
			return NewError(CodeValidation, "传感器名称 %q 重复", name)
		}
		if locations[locationKey] {
			return NewError(CodeValidation, "传感器位置 %q 重复", location)
		}
		names[nameKey], locations[locationKey] = true, true
	}
	return nil
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func NormalizeTargetName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(name)), " "))
}

type TargetBatchIssue struct {
	ItemIndex            int      `json:"itemIndex"`
	Code                 string   `json:"code"`
	Fields               []string `json:"fields"`
	ExistingTargetID     string   `json:"existingTargetId,omitempty"`
	ConflictingItemIndex int      `json:"conflictingItemIndex,omitempty"`
	Message              string   `json:"message"`
}

func ValidateTargetBatch(targets, existing []ProtectedTarget) []TargetBatchIssue {
	issues := []TargetBatchIssue{}
	existingKeys := map[string]string{}
	for _, t := range existing {
		existingKeys[t.TargetType+"\x00"+NormalizeTargetName(t.Name)] = t.TargetID
	}
	batchKeys := map[string]int{}
	for i, t := range targets {
		item := i + 1
		if err := ValidateTarget(t); err != nil {
			issues = append(issues, TargetBatchIssue{ItemIndex: item, Code: "invalid_target", Message: err.Error()})
		}
		key := t.TargetType + "\x00" + NormalizeTargetName(t.Name)
		if id := existingKeys[key]; id != "" {
			issues = append(issues, TargetBatchIssue{ItemIndex: item, Code: "duplicate_existing_target", Fields: []string{"targetType", "name"}, ExistingTargetID: id, Message: "与案卷既有保护对象重复"})
		}
		if first := batchKeys[key]; first != 0 {
			issues = append(issues, TargetBatchIssue{ItemIndex: item, Code: "duplicate_batch_target", Fields: []string{"targetType", "name"}, ConflictingItemIndex: first, Message: "与本批次保护对象重复"})
		} else if NormalizeTargetName(t.Name) != "" {
			batchKeys[key] = item
		}
	}
	return issues
}
