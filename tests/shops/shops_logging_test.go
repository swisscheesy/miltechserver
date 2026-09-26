package shops_test

import (
	"bytes"
	"context"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/members"
	"miltechserver/api/shops/members/invites"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"strings"
	"testing"
)

type loggingInviteRepo struct {
	invites.Repository
	code model.ShopInviteCodes
}

func (r *loggingInviteRepo) CreateInviteCode(_ *bootstrap.User, code model.ShopInviteCodes) (*model.ShopInviteCodes, error) {
	r.code = code
	return &r.code, nil
}
func (r *loggingInviteRepo) GetInviteCodeByCode(_ string) (*model.ShopInviteCodes, error) {
	return &r.code, nil
}

type loggingMemberRepo struct {
	members.Repository
	joined bool
}

func (r *loggingMemberRepo) AddMemberToShop(_ *bootstrap.User, _, _ string) error {
	r.joined = true
	return nil
}

type loggingAuth struct {
	shared.ShopAuthorization
	member bool
}

func (a loggingAuth) IsUserMemberOfShop(_ *bootstrap.User, _ string) (bool, error) {
	return a.member, nil
}

func TestLoggingInviteAndJoin(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	repo := &loggingInviteRepo{}
	user := &bootstrap.User{UserID: "logging-user"}
	code, err := invites.NewService(repo, loggingAuth{member: true}).GenerateInviteCode(context.Background(), user, "logging-shop")
	if err != nil {
		t.Fatal(err)
	}
	if code.Code == "" {
		t.Fatal("missing invite code in unchanged response")
	}
	if strings.Contains(output.String(), code.Code) {
		t.Error("generated secret leaked")
	}
	memberRepo := &loggingMemberRepo{}
	repo.code.Code = "sentinel-invite-secret"
	if err := members.NewService(memberRepo, repo, loggingAuth{}).JoinShopViaInviteCode(context.Background(), user, repo.code.Code); err != nil {
		t.Fatal(err)
	}
	if !memberRepo.joined {
		t.Fatal("join did not execute")
	}
	if strings.Contains(output.String(), repo.code.Code) {
		t.Error("join secret leaked")
	}
	for _, safe := range []string{"invite_generated", "shop_joined", "logging-shop", "success"} {
		if !strings.Contains(output.String(), safe) {
			t.Errorf("missing diagnostic %s", safe)
		}
	}
}
