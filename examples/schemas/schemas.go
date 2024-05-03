package schemas

import (
	"github.com/guregu/null/v5"
)

type Org struct {
	ID   uint `dba:"pk"`
	Code string
	Name string
}

type User struct {
	ID          uint        `dba:"pk"`
	Nickname    string      `dba:"desc=用户昵称"`
	Profile     *Profile    `dba:"desc=用户档案;rel=HAS_ONE,ID->UserID"`
	Accounts    []*Account  `dba:"desc=用户账户;rel=HAS_MANY,ID->UserID"`
	OrgID       uint        `dba:"desc=所属组织ID"`
	Org         *Org        `dba:"desc=所属组织;rel=REF_ONE,OrgID->ID"`
	Roles       []*Role     `dba:"desc=拥有角色;rel=REF_MANY,user_role_ref(id->user_id,id->role_id)"` // 直接对表
	Departments []*UserDept `dba:"desc=所属部门;rel=REF_MANY,UserDept(ID->UserID,ID->DeptID)"`        // 对结构体
}

type Profile struct {
	ID       uint   `dba:"pk"`
	UserID   uint   `dba:"desc=关联用户ID"`
	User     *User  `dba:"desc=关联用户;rel=REF_ONE,UserID->ID"`
	RealName string `dba:"desc=真实姓名"`
}

type Account struct {
	ID       uint   `dba:"pk"`
	UserID   uint   `dba:"desc=关联用户ID"`
	User     *User  `dba:"desc=关联用户;rel=REF_ONE,UserID->ID"`
	Type     string `dba:"desc=账户类型"` // 如代码仓库账号、业务系统账号、邮箱系统账号等等
	Username string `dba:"desc=账户名称"`
	Password string `dba:"desc=账户密码"`
}

type Dept struct {
	ID   uint   `dba:"pk"`
	Name string `dba:"desc=部门名称"`
}

type UserDept struct {
	UserID uint `dba:"desc=关联用户ID;pk"`
	DeptID uint `dba:"desc=关联部门ID;pk"`

	User   *User `dba:"desc=关联用户;rel=REF_ONE,UserID->ID"`
	Dept   *Dept `dba:"desc=关联部门;rel=REF_ONE,DeptID->ID"`
	IsMain null.Bool
}

type Role struct {
	ID          uint          `dba:"pk"`
	Name        string        `dba:"desc=角色名称"`
	Permissions []*Permission `dba:"desc=角色权限;rel=REF_MANY,role_permission_ref(id->role_id,id->permission_id)"` // 直接对表
}

type Permission struct {
	ID   uint   `dba:"pk"`
	Code string `dba:"desc=权限代码"`
	Name string `dba:"desc=权限名称"`
}
