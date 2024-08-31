package examples

import (
	"github.com/guregu/null/v5"
	"time"
)

type User struct {
	ID          uint        `dba:"pk;incr"`
	Nickname    string      `dba:"title=用户昵称"`
	Profile     *Profile    `dba:"title=用户档案;rel=HAS_ONE,ID->UserID"`
	Accounts    []*Account  `dba:"title=用户账户;rel=HAS_MANY,ID->UserID"`
	OrgID       uint        `dba:"title=所属组织ID"`
	Org         *Org        `dba:"title=所属组织;rel=REF_ONE,OrgID->ID"`
	Roles       []*Role     `dba:"title=拥有角色;rel=REF_MANY,user_role_ref(id->user_id,id->role_id)"` // 直接对表
	Departments []*UserDept `dba:"title=所属部门;rel=HAS_MANY,UserDept(ID->UserID,ID->DeptID)"`        // 对结构体
}

type Profile struct {
	ID       uint   `dba:"pk;incr"`
	UserID   uint   `dba:"title=关联用户ID"`
	User     *User  `dba:"title=关联用户;rel=REF_ONE,UserID->ID"`
	RealName string `dba:"title=真实姓名"`
}

type Account struct {
	ID       uint   `dba:"pk;incr"`
	UserID   uint   `dba:"title=关联用户ID"`
	User     *User  `dba:"title=关联用户;rel=REF_ONE,UserID->ID"`
	Type     string `dba:"title=账户类型"` // 如代码仓库账号、业务系统账号、邮箱系统账号等等
	Username string `dba:"title=账户名称"`
	Password string `dba:"title=账户密码"`
	Remark   string `dba:"title=备注;type=text"`
}

type Org struct {
	Model
	Code string
	Name string
}

type Dept struct {
	ID   uint   `dba:"pk;incr"`
	Name string `dba:"title=部门名称"`
}

type UserDept struct {
	UserID uint `dba:"title=关联用户ID;pk"`
	DeptID uint `dba:"title=关联部门ID;pk"`

	User   *User     `dba:"title=关联用户;rel=REF_ONE,UserID->ID"`
	Dept   *Dept     `dba:"title=关联部门;rel=REF_ONE,DeptID->ID"`
	IsMain null.Bool `dba:"title=是否主部门"`
}

type Role struct {
	Model
	Name        string        `dba:"title=角色名称"`
	Permissions []*Permission `dba:"title=角色权限;rel=REF_MANY,role_permission_ref(id->role_id,id->permission_id)"` // 直接对表
	Others      []string
}

type Permission struct {
	ID   uint   `dba:"pk;incr"`
	Code string `dba:"title=权限代码"`
	Name string `dba:"title=权限名称"`
}

type Model struct {
	ID        uint        `dba:"pk;incr"`
	CreatedAt time.Time   `dba:"title=创建时间"`
	UpdatedAt time.Time   `dba:"title=最后修改时间"`
	DeletedAt *time.Time  `dba:"title=逻辑删除时间;null"`
	Remark    null.String `dba:"title=备注;type=text"`
}
