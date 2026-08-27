package selfcheck

import (
	"context"
	"fmt"
	"net/http"

	"blast-permit/internal/application"
	"blast-permit/internal/domain"
)

func runFlow(ctx context.Context, c *client) error {
	designerRole, reviewerRole, safetyRole := "designer", "reviewer", "safety_officer"
	var created application.CaseResponse
	if err := c.request(ctx, http.MethodPost, "/api/v1/cases", designerRole, "张工", map[string]any{"siteName": "东岭地下厂房", "workZone": "引水洞 K2+300", "idempotencyKey": "self-create-001"}, http.StatusCreated, &created); err != nil {
		return err
	}
	path := "/api/v1/cases/" + created.CaseID
	targets := map[string]any{"expectedVersion": created.Version, "idempotencyKey": "self-targets-001", "targets": []map[string]any{{"targetType": "building", "name": "通风竖井", "distanceMeters": 20, "allowedPpvMmPerSec": 10, "baselinePpvMmPerSec": 0.3, "measurementNote": "静态采集 30 分钟"}, {"targetType": "pipeline", "name": "DN600 输水管", "distanceMeters": 35, "allowedPpvMmPerSec": 12, "baselinePpvMmPerSec": 0.2, "measurementNote": "管壁基线采样"}, {"targetType": "equipment", "name": "主变压器", "distanceMeters": 50, "allowedPpvMmPerSec": 8, "baselinePpvMmPerSec": 0.1, "measurementNote": "设备基础三向采样"}}}
	var changed application.AddTargetsResponse
	if err := c.request(ctx, http.MethodPost, path+"/targets", designerRole, "张工", targets, http.StatusOK, &changed); err != nil {
		return err
	}
	var targetReplay application.AddTargetsResponse
	if err := c.request(ctx, http.MethodPost, path+"/targets", designerRole, "张工", targets, http.StatusOK, &targetReplay); err != nil {
		return err
	}
	if len(targetReplay.Targets) != len(changed.Targets) || targetReplay.Targets[0].TargetID != changed.Targets[0].TargetID {
		return fmt.Errorf("保护对象幂等重放未返回首次登记结果")
	}
	duplicate := map[string]any{"expectedVersion": changed.Version, "idempotencyKey": "self-targets-duplicate", "targets": []map[string]any{{"targetType": "building", "name": "  通风竖井  ", "distanceMeters": 20, "allowedPpvMmPerSec": 10, "baselinePpvMmPerSec": 0.3, "measurementNote": "重复对象不应写入"}}}
	if err := c.request(ctx, http.MethodPost, path+"/targets", designerRole, "张工", duplicate, http.StatusConflict, nil); err != nil {
		return fmt.Errorf("保护对象业务重复未被拒绝: %w", err)
	}
	var baselinePrecheck domain.BaselinePrecheck
	if err := c.request(ctx, http.MethodGet, path+"/baseline/precheck", "", "", nil, http.StatusOK, &baselinePrecheck); err != nil {
		return err
	}
	if !baselinePrecheck.Ready || baselinePrecheck.ControlTargetID == "" {
		return fmt.Errorf("完整基线未通过只读预检")
	}
	var targetAudit struct {
		Events []domain.AuditEvent `json:"events"`
	}
	if err := c.request(ctx, http.MethodGet, path+"/audit", "", "", nil, http.StatusOK, &targetAudit); err != nil {
		return err
	}
	if len(targetAudit.Events) != 2 {
		return fmt.Errorf("保护对象重放、冲突或基线预检错误追加了审计事件")
	}
	stale := map[string]any{"expectedVersion": created.Version, "idempotencyKey": "self-stale-001"}
	if err := c.request(ctx, http.MethodPost, path+"/baseline/complete", designerRole, "张工", stale, http.StatusConflict, nil); err != nil {
		return fmt.Errorf("陈旧版本未被拒绝: %w", err)
	}
	var baseline application.CaseResponse
	if err := c.request(ctx, http.MethodPost, path+"/baseline/complete", designerRole, "张工", map[string]any{"expectedVersion": changed.Version, "idempotencyKey": "self-baseline-001"}, http.StatusOK, &baseline); err != nil {
		return err
	}
	hazard := map[string]any{"expectedVersion": baseline.Version, "idempotencyKey": "self-revision-001", "holePattern": "3.0m×2.5m 梅花形", "maxChargePerDelayKg": 100, "delaySequenceMs": []int{0, 25, 50}, "initiationDirection": "由保护对象向掌子面", "propagationK": 100, "propagationAlpha": 1.6}
	var assessed application.RevisionResponse
	if err := c.request(ctx, http.MethodPost, path+"/revisions", designerRole, "张工", hazard, http.StatusCreated, &assessed); err != nil {
		return err
	}
	if assessed.State != domain.StateRemediationRequired || len(assessed.Assessment.BlockingFindingIDs) == 0 {
		return fmt.Errorf("高药量修订未生成阻断项")
	}
	if len(assessed.Assessment.AllowedCharges) != len(changed.Targets) || assessed.Assessment.ControlTargetID == "" {
		return fmt.Errorf("评估未生成逐目标允许药量和控制目标")
	}
	corrected := map[string]any{"expectedVersion": assessed.Version, "idempotencyKey": "self-remedy-001", "holePattern": "2.0m×1.8m 梅花形", "maxChargePerDelayKg": 1, "delaySequenceMs": []int{0, 25, 50, 75}, "initiationDirection": "背离保护对象", "propagationK": 100, "propagationAlpha": 1.6, "remediationNote": "降低单段药量并细化延期"}
	resolutions := make([]map[string]any, 0, len(assessed.Assessment.BlockingFindingIDs))
	for _, findingID := range assessed.Assessment.BlockingFindingIDs {
		resolutions = append(resolutions, map[string]any{"findingId": findingID, "handlingNote": "降低药量并复算该阻断项"})
	}
	corrected["findingResolutions"] = resolutions
	var remedied application.RevisionResponse
	if err := c.request(ctx, http.MethodPost, path+"/remediations", designerRole, "张工", corrected, http.StatusCreated, &remedied); err != nil {
		return err
	}
	if remedied.State != domain.StateReviewReady {
		return fmt.Errorf("整改后状态为 %s", remedied.State)
	}
	var auditBefore struct {
		Events []domain.AuditEvent `json:"events"`
	}
	if err := c.request(ctx, http.MethodGet, path+"/audit", "", "", nil, http.StatusOK, &auditBefore); err != nil {
		return err
	}
	var replay application.RevisionResponse
	if err := c.request(ctx, http.MethodPost, path+"/remediations", designerRole, "张工", corrected, http.StatusCreated, &replay); err != nil {
		return err
	}
	if replay.Revision.RevisionID != remedied.Revision.RevisionID {
		return fmt.Errorf("幂等请求未恢复首次响应")
	}
	var auditAfter struct {
		Events []domain.AuditEvent `json:"events"`
	}
	if err := c.request(ctx, http.MethodGet, path+"/audit", "", "", nil, http.StatusOK, &auditAfter); err != nil {
		return err
	}
	if len(auditBefore.Events) != len(auditAfter.Events) {
		return fmt.Errorf("幂等重放新增了审计事件")
	}
	var file domain.CaseFile
	if err := c.request(ctx, http.MethodGet, path, "", "", nil, http.StatusOK, &file); err != nil {
		return err
	}
	points := make([]map[string]any, 0, len(file.Targets))
	for i, t := range file.Targets {
		points = append(points, map[string]any{"name": fmt.Sprintf("S%d", i+1), "targetId": t.TargetID, "location": fmt.Sprintf("保护对象迎爆侧基础测点 %d", i+1)})
	}
	reject := map[string]any{"expectedVersion": remedied.Version, "idempotencyKey": "self-review-reject", "sensorPoints": points, "sampleRateHz": 4000, "triggerPpvMmPerSec": 7, "evacuationRule": "任一点达到阈值立即撤离", "remainingRisk": "设备侧传感器需加固", "decision": "reject", "reasons": []map[string]any{{"category": "charge", "description": "控制目标药量余量仍需扩大", "parameter": "maxChargePerDelayKg", "requiredChange": "继续降低单段最大药量"}}}
	var rejected application.ReviewResponse
	if err := c.request(ctx, http.MethodPost, path+"/reviews", reviewerRole, "李复核", reject, http.StatusOK, &rejected); err != nil {
		return err
	}
	if rejected.State != domain.StateChangesRequired {
		return fmt.Errorf("退回后状态错误")
	}
	emptyRevision := map[string]any{"expectedVersion": rejected.Version, "idempotencyKey": "self-revision-empty", "holePattern": "2.0m×1.8m 梅花形", "maxChargePerDelayKg": 1, "delaySequenceMs": []int{0, 25, 50, 75}, "initiationDirection": "背离保护对象", "propagationK": 100, "propagationAlpha": 1.6, "remediationNote": "只改说明不得形成新修订"}
	if err := c.request(ctx, http.MethodPost, path+"/revisions", designerRole, "张工", emptyRevision, http.StatusConflict, nil); err != nil {
		return fmt.Errorf("空修订未被拒绝: %w", err)
	}
	newRevision := map[string]any{"expectedVersion": rejected.Version, "idempotencyKey": "self-revision-002", "holePattern": "2.0m×1.8m 梅花形", "maxChargePerDelayKg": 0.8, "delaySequenceMs": []int{0, 25, 50, 75}, "initiationDirection": "背离保护对象", "propagationK": 100, "propagationAlpha": 1.6, "remediationNote": "按复核意见加固测点并再降药量"}
	var revised application.RevisionResponse
	if err := c.request(ctx, http.MethodPost, path+"/revisions", designerRole, "张工", newRevision, http.StatusCreated, &revised); err != nil {
		return err
	}
	if revised.Revision.RevisionNumber <= remedied.Revision.RevisionNumber {
		return fmt.Errorf("退回后未形成新修订")
	}
	if revised.Revision.ParentRevisionID != remedied.Revision.RevisionID || len(revised.Revision.Diff.Changes) != 1 || revised.Revision.Diff.Changes[0].Field != "maxChargePerDelayKg" || len(revised.Revision.Diff.AffectedTargetIDs) != len(file.Targets) {
		return fmt.Errorf("药量修订未形成正确的父修订、字段差异或影响范围")
	}
	approveMissing := map[string]any{"expectedVersion": revised.Version, "idempotencyKey": "self-review-unconfirmed", "sensorPoints": points, "sampleRateHz": 4000, "triggerPpvMmPerSec": 7, "evacuationRule": "任一点达到阈值立即撤离并封锁警戒区", "remainingRisk": "残余风险可由现场停工阈值控制", "decision": "approve"}
	if err := c.request(ctx, http.MethodPost, path+"/reviews", reviewerRole, "李复核", approveMissing, http.StatusConflict, nil); err != nil {
		return fmt.Errorf("未确认退回原因的批准未被拒绝: %w", err)
	}
	approve := map[string]any{"expectedVersion": revised.Version, "idempotencyKey": "self-review-approve", "sensorPoints": points, "sampleRateHz": 4000, "triggerPpvMmPerSec": 7, "evacuationRule": "任一点达到阈值立即撤离并封锁警戒区", "remainingRisk": "残余风险可由现场停工阈值控制", "decision": "approve", "reasonResolutions": []map[string]any{{"reasonId": rejected.Review.Reasons[0].ReasonID, "confirmed": true, "note": "已核对药量差异与复算结果"}}}
	var approved application.CaseResponse
	if err := c.request(ctx, http.MethodPost, path+"/reviews", reviewerRole, "李复核", approve, http.StatusOK, &approved); err != nil {
		return err
	}
	if approved.State != domain.StateApproved {
		return fmt.Errorf("批准状态错误")
	}
	var permitPrecheck domain.PermitPrecheck
	if err := c.request(ctx, http.MethodGet, path+"/permit/precheck", "", "", nil, http.StatusOK, &permitPrecheck); err != nil {
		return err
	}
	if !permitPrecheck.Ready || len(permitPrecheck.Components) != 4 {
		return fmt.Errorf("许可证据预检未形成四类完整组件")
	}
	var issued application.PermitResponse
	if err := c.request(ctx, http.MethodPost, path+"/permit", safetyRole, "王安全", map[string]any{"expectedVersion": approved.Version, "idempotencyKey": "self-permit-001", "validHours": 8}, http.StatusCreated, &issued); err != nil {
		return err
	}
	if issued.State != domain.StateFrozen {
		return fmt.Errorf("许可签发后未冻结")
	}
	if issued.Permit.EvidenceDigest != permitPrecheck.EvidenceDigest {
		return fmt.Errorf("签发许可未复用证据预检摘要")
	}
	var verified application.VerificationResponse
	if err := c.request(ctx, http.MethodGet, "/api/v1/permits/"+issued.Permit.PermitNumber+"/verify", "", "", nil, http.StatusOK, &verified); err != nil {
		return err
	}
	if !verified.Valid || verified.EvidenceDigest != issued.Permit.EvidenceDigest {
		return fmt.Errorf("许可验真失败")
	}
	frozenWrite := map[string]any{"expectedVersion": issued.Version, "idempotencyKey": "self-frozen-write", "targets": []map[string]any{{"targetType": "building", "name": "冻结后新增对象", "distanceMeters": 80, "allowedPpvMmPerSec": 10, "baselinePpvMmPerSec": 0.1, "measurementNote": "不应写入"}}}
	if err := c.request(ctx, http.MethodPost, path+"/targets", designerRole, "张工", frozenWrite, http.StatusConflict, nil); err != nil {
		return fmt.Errorf("冻结保护失败: %w", err)
	}
	return nil
}
