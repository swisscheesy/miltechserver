package lists

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/equipment_services/core"
	shopcore "miltechserver/api/shops/core"
	"miltechserver/api/shops/members"
	"miltechserver/api/shops/shared"
	"miltechserver/api/shops/vehicles/notifications"
	"miltechserver/bootstrap"
	"strings"
	"testing"
)

type dependencyConnector struct{ state *dependencyConn }

func (c dependencyConnector) Connect(context.Context) (driver.Conn, error) { return c.state, nil }
func (c dependencyConnector) Driver() driver.Driver                        { return dependencyDriver{} }

type dependencyDriver struct{}

func (dependencyDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }

type dependencyConn struct {
	admin                                 bool
	memberCount                           int
	dependency                            bool
	checked, detached, deleted, committed bool
	failDelete                            bool
	failDetach                            bool
	missingList                           bool
	queries                               []string
	lockedLists                           []string
}

func (*dependencyConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (*dependencyConn) Close() error                { return nil }
func (c *dependencyConn) Begin() (driver.Tx, error) { return c, nil }
func (c *dependencyConn) Commit() error             { c.committed = true; return nil }
func (*dependencyConn) Rollback() error             { return nil }
func (c *dependencyConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	c.queries = append(c.queries, q)
	if strings.Contains(q, "FROM shop_lists") && strings.Contains(q, "FOR UPDATE") {
		c.lockedLists = append(c.lockedLists, args[0].Value.(string))
		if c.missingList {
			return &dependencyRows{read: true}, nil
		}
	}

	var values []driver.Value
	switch {
	case strings.Contains(q, "EXISTS") && strings.Contains(q, "equipment_services"):
		c.checked = true
		values = []driver.Value{c.dependency}
	case strings.HasPrefix(q, "SELECT admin_only_lists"):
		values = []driver.Value{false}
	case strings.HasPrefix(q, "SELECT count(*) FROM shop_members"):
		values = []driver.Value{int64(c.memberCount)}
	case strings.HasPrefix(q, "SELECT role"):
		values = []driver.Value{"member"}
		if c.admin {
			values = []driver.Value{"admin"}
		}
	case strings.HasPrefix(q, "SELECT list_id,created_by"):
		values = []driver.Value{"old-list", "user"}
	case strings.HasPrefix(q, "SELECT attached_shop_list"):
		values = []driver.Value{"old-list"}
	case strings.HasPrefix(q, "SELECT shop_id,created_by"):
		values = []driver.Value{"shop", "user"}
	case strings.HasPrefix(q, "SELECT shop_id"):
		values = []driver.Value{"shop"}
	default:
		return nil, fmt.Errorf("unexpected query %s", q)
	}
	return &dependencyRows{values: values}, nil
}
func (c *dependencyConn) ExecContext(_ context.Context, q string, _ []driver.NamedValue) (driver.Result, error) {
	if strings.Contains(q, "UPDATE") && strings.Contains(q, "shop_vehicle_notifications") {
		c.detached = true
		if c.failDetach {
			return nil, errors.New("injected detach failure")
		}
		return driver.RowsAffected(1), nil
	}
	if strings.Contains(q, "DELETE") {
		c.deleted = true
		if c.failDelete {
			return nil, errors.New("injected failure")
		}
		return driver.RowsAffected(1), nil
	}
	return nil, fmt.Errorf("unexpected exec %s", q)
}

type dependencyRows struct {
	values []driver.Value
	read   bool
}

func (r *dependencyRows) Columns() []string {
	v := make([]string, len(r.values))
	for i := range v {
		v[i] = fmt.Sprint(i)
	}
	return v
}
func (*dependencyRows) Close() error { return nil }
func (r *dependencyRows) Next(dest []driver.Value) error {
	if r.read {
		return io.EOF
	}
	copy(dest, r.values)
	r.read = true
	return nil
}

func TestListDependenciesBlockBeforeAnyWrite(t *testing.T) {
	c := &dependencyConn{dependency: true}
	db := sql.OpenDB(dependencyConnector{c})
	defer db.Close()
	err := NewRepository(db).DeleteShopList(&bootstrap.User{UserID: "user"}, "list")
	if err == nil || err.Error() != "list is in use" {
		t.Fatalf("want list conflict, got %v", err)
	}
	if !c.checked || c.deleted || c.detached || c.committed {
		t.Fatalf("dependency mutated state: %+v", c)
	}
}
func TestListDependenciesDetachBeforeSuccessfulDelete(t *testing.T) {
	c := &dependencyConn{}
	db := sql.OpenDB(dependencyConnector{c})
	defer db.Close()
	if err := NewRepository(db).DeleteShopList(&bootstrap.User{UserID: "user"}, "list"); err != nil {
		t.Fatal(err)
	}
	if !c.checked || !c.detached || !c.deleted || !c.committed {
		t.Fatalf("missing preservation operation: %+v", c)
	}
}
func TestListDependenciesDeleteFailureDoesNotCommit(t *testing.T) {
	c := &dependencyConn{failDelete: true}
	db := sql.OpenDB(dependencyConnector{c})
	defer db.Close()
	if err := NewRepository(db).DeleteShopList(&bootstrap.User{UserID: "user"}, "list"); err == nil {
		t.Fatal("want failure")
	}
	if c.committed {
		t.Fatal("committed failure")
	}
}

func TestListDependenciesDetachFailureDoesNotDelete(t *testing.T) {
	c := &dependencyConn{failDetach: true}
	db := sql.OpenDB(dependencyConnector{c})
	defer db.Close()
	if err := NewRepository(db).DeleteShopList(&bootstrap.User{UserID: "user"}, "list"); err == nil {
		t.Fatal("want detach failure")
	}
	if c.deleted || c.committed {
		t.Fatal("delete or commit after detach failure")
	}
}

func TestListDependenciesWritersRejectDeletedAttachment(t *testing.T) {
	for _, name := range []string{"create-service", "update-service", "create-notification", "update-notification"} {
		t.Run(name, func(t *testing.T) {
			c := &dependencyConn{missingList: true}
			db := sql.OpenDB(dependencyConnector{c})
			defer db.Close()
			user := &bootstrap.User{UserID: "user"}
			list := "list"
			var err error
			switch name {
			case "create-service":
				_, err = core.NewRepository(db).Create(user, model.EquipmentServices{ShopID: "shop", EquipmentID: "vehicle", ListID: list})
			case "update-service":
				_, err = core.NewRepository(db).Update(user, model.EquipmentServices{ID: "service", ListID: list})
			case "create-notification":
				_, err = notifications.NewRepository(db).CreateVehicleNotification(user, model.ShopVehicleNotifications{ShopID: "shop", VehicleID: "vehicle", AttachedShopList: &list})
			case "update-notification":
				err = notifications.NewRepository(db).UpdateVehicleNotification(user, notifications.VehicleNotificationUpdate{Notification: model.ShopVehicleNotifications{ID: "notification"}, AttachedShopListSet: true, AttachedShopList: &list})
			}
			if err == nil {
				t.Fatal("missing referenced list was accepted")
			}
			if len(c.lockedLists) == 0 {
				t.Fatalf("writer bypassed list lock: %v", err)
			}
			shop, member, listLock := -1, -1, -1
			for i, q := range c.queries {
				if strings.HasPrefix(q, "SELECT admin_only_lists") {
					shop = i
				}
				if strings.HasPrefix(q, "SELECT role") {
					member = i
				}
				if strings.Contains(q, "FROM shop_lists") && strings.Contains(q, "FOR UPDATE") {
					listLock = i
				}
			}
			if !(shop >= 0 && member > shop && listLock > member) {
				t.Fatalf("wrong lock order: %v", c.queries)
			}
			if c.committed {
				t.Fatal("committed failed attachment")
			}
		})
	}
}

func TestListDependenciesLocksSortedUniqueLists(t *testing.T) {
	c := &dependencyConn{}
	db := sql.OpenDB(dependencyConnector{c})
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, _, err := shared.LockShopMutation(context.Background(), tx, "shop", "user"); err != nil {
		t.Fatal(err)
	}
	if err := shared.LockReferencedLists(context.Background(), tx, "shop", "z", "a", "z", ""); err != nil {
		t.Fatal(err)
	}
	if strings.Join(c.lockedLists, ",") != "a,z" {
		t.Fatalf("wrong list ordering: %v", c.lockedLists)
	}
}

func TestListDependenciesAggregateDeletionRequiresMembershipLock(t *testing.T) {
	for _, name := range []string{"shop", "last-member"} {
		t.Run(name, func(t *testing.T) {
			c := &dependencyConn{admin: true, memberCount: 1}
			db := sql.OpenDB(dependencyConnector{c})
			defer db.Close()
			user := &bootstrap.User{UserID: "user"}
			var err error
			if name == "shop" {
				err = shopcore.NewRepository(db, nil, &bootstrap.Env{}).DeleteShop(user, "shop")
			} else {
				err = members.NewRepository(db, nil, &bootstrap.Env{}).DeleteShop(user, "shop")
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(c.queries) < 2 || !strings.HasPrefix(c.queries[0], "SELECT admin_only_lists") || !strings.HasPrefix(c.queries[1], "SELECT role") {
				t.Fatalf("aggregate bypassed membership locks: %v", c.queries)
			}
			if !c.deleted || !c.committed || c.checked {
				t.Fatalf("aggregate must bypass single-list dependency check: %+v", c)
			}
		})
	}
}
func TestListDependenciesLastMemberRechecksCount(t *testing.T) {
	c := &dependencyConn{memberCount: 2}
	db := sql.OpenDB(dependencyConnector{c})
	defer db.Close()
	if err := members.NewRepository(db, nil, &bootstrap.Env{}).DeleteShop(&bootstrap.User{UserID: "user"}, "shop"); err == nil {
		t.Fatal("stale last-member count accepted")
	}
	if c.deleted || c.committed {
		t.Fatal("deleted shop after another member joined")
	}
}
