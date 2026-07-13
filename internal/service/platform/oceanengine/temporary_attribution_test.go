package oceanengine

import "testing"

func TestQianchuanTemporaryAttributionRulesResolve(t *testing.T) {
	rules := qianchuanTemporaryAttributionRules{
		1700642741413895: {
			{
				AdvertiserID:       1700642741413895,
				TargetTraderID:     22,
				TargetCompanyID:    3,
				TargetDepartmentID: 8,
				EffectiveStartDate: "2026-07-13",
			},
		},
	}
	original := qianchuanOrgMapping{
		AccountID:      11,
		CompanyID:      1,
		DepartmentID:   2,
		TraderID:       9,
		OrganizationID: 1,
	}

	before, matched := rules.resolve(1700642741413895, "2026-07-12", original)
	if matched || before != original {
		t.Fatalf("rule should not apply before start date: matched=%v mapping=%+v", matched, before)
	}

	after, matched := rules.resolve(1700642741413895, "2026-07-13 00:00:00", original)
	if !matched {
		t.Fatal("expected rule to apply on start date")
	}
	if after.AccountID != original.AccountID || after.TraderID != 22 || after.CompanyID != 3 || after.DepartmentID != 8 || after.OrganizationID != 3 {
		t.Fatalf("unexpected temporary attribution mapping: %+v", after)
	}
}

func TestQianchuanTemporaryAttributionRulesEndDate(t *testing.T) {
	rules := qianchuanTemporaryAttributionRules{
		1: {{
			AdvertiserID:       1,
			TargetTraderID:     2,
			EffectiveStartDate: "2026-07-01",
			EffectiveEndDate:   "2026-07-10",
		}},
	}

	if _, matched := rules.resolve(1, "2026-07-10", qianchuanOrgMapping{}); !matched {
		t.Fatal("expected end date to be inclusive")
	}
	if _, matched := rules.resolve(1, "2026-07-11", qianchuanOrgMapping{}); matched {
		t.Fatal("rule should not apply after end date")
	}
}
