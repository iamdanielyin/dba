package examples

import "time"

type SubscriptionPlan struct {
}

type Tenant struct {
	ID       uint   `dba:"title=主键ID"`
	Name     string `dba:"title=租户名称"`
	IsActive *bool  `dba:"title=是否激活"`
}

type User struct {
	ID             uint         `dba:"title=主键ID"`
	Username       string       `dba:"title=用户名"`
	Password       string       `dba:"title=密码"`
	ProfileID      *uint        `dba:"title=用户详细信息ID"`
	Profile        *UserProfile `dba:"title=用户详细信息;rel=HAS_ONE,ID->UserID"`
	Addresses      []*Address   `dba:"title=收货地址;rel=HAS_MANY,ID->UserID"`
	DefaultAddress *Address     `dba:"title=默认收货地址;rel=HAS_ONE,DefaultAddressID->UserID;rel_filter=IsDefault:true"`
	TenantID       uint         `dba:"title=所属租户ID"`
	Tenant         *Tenant      `dba:"title=所属租户;rel=REF_ONE,TenantID->ID"`
	Tags           []*Tag       `dba:"title=已有标签;rel=REF_MANY,user_tag_refers(id->User_id,id->tag_id)"` // 直接对表
	Groups         []*UserGroup `dba:"title=已加群组;rel=HAS_MANY,UserGroup(ID->UserID,ID->GroupID)"`
}

type UserProfile struct {
	ID       uint   `dba:"name=id;title=详细信息ID"`
	UserID   uint   `dba:"name=user_id;title=用户ID"`
	Birthday string `dba:"name=birthday;title=生日（YYYY-MM-DD格式）"`
	Gender   string `dba:"name=gender;title=性别（男、女）"`
}

type Address struct {
	ID          uint   `dba:"title=主键ID"`
	UserID      uint   `dba:"title=用户ID"`
	User        *User  `dba:"title=用户ID;rel=REF_ONE,UserID->ID;"`
	Name        string `dba:"title=收货人姓名"`
	Phone       string `dba:"title=收货人电话"`
	Province    string `dba:"title=省份"`
	City        string `dba:"title=城市"`
	District    string `dba:"title=区/县"`
	AddressLine string `dba:"title=详细地址"`
	PostalCode  string `dba:"title=邮政编码"`
	IsDefault   *bool  `dba:"title=是否默认地址"`
}

type Group struct {
	ID        uint      `dba:"title=群组ID"`
	Name      string    `dba:"title=群组名称"`
	CreatedAt time.Time `dba:"title=创建时间"`
}

type Tag struct {
	ID   uint   `dba:"title=标签ID"`
	Name string `dba:"title=标签名称"`
}

type UserGroup struct {
	UserID   uint      `dba:"title=用户ID"`
	GroupID  uint      `dba:"title=群组ID"`
	JoinedAt time.Time `dba:"title=加入时间"`
}
