package examples

import "time"

type Tenant struct {
	ID       uint   `dba:"title=主键ID;pk;incr"`
	Name     string `dba:"title=租户名称;required"`
	IsActive *bool  `dba:"title=是否激活"`
}

type User struct {
	ID             uint         `dba:"title=主键ID;pk;incr"`
	Username       string       `dba:"title=用户名;required"`
	Password       string       `dba:"title=密码"`
	ProfileID      *uint        `dba:"title=用户详细信息ID"`
	Profile        *UserProfile `dba:"title=用户详细信息;rel=HAS_ONE,ID->UserID"`
	Addresses      []*Address   `dba:"title=收货地址;rel=HAS_MANY,ID->UserID"`
	DefaultAddress *Address     `dba:"title=默认收货地址;rel=HAS_ONE,DefaultAddressID->UserID;rel_filter=IsDefault:true"`
	TenantID       uint         `dba:"title=所属租户ID"`
	Tenant         *Tenant      `dba:"title=所属租户;rel=REF_ONE,TenantID->ID"`
	Tags           []*Tag       `dba:"title=已有标签;rel=REF_MANY,user_tag_refers(id->user_id,id->tag_id)"` // 直接对表
	Groups         []*UserGroup `dba:"title=已加群组;rel=REF_MANY,UserGroup(ID->UserID,ID->GroupID)"`
}

type UserProfile struct {
	UserID   uint   `dba:"name=user_id;title=用户ID;pk"`
	Birthday string `dba:"name=birthday;title=生日（YYYY-MM-DD格式）"`
	Gender   string `dba:"name=gender;title=性别（男、女）"`
}

type Address struct {
	ID         uint   `dba:"title=主键ID;pk;incr"`
	UserID     uint   `dba:"title=用户ID;required"`
	User       *User  `dba:"title=用户;rel=REF_ONE,UserID->ID;"`
	Name       string `dba:"title=收货人姓名;required"`
	Phone      string `dba:"title=收货人电话;required"`
	Province   string `dba:"title=省份;required"`
	City       string `dba:"title=城市;required"`
	District   string `dba:"title=区/县;required"`
	Address    string `dba:"title=详细地址;required"`
	PostalCode string `dba:"title=邮政编码"`
	IsDefault  *bool  `dba:"title=是否默认地址"`
}

type Group struct {
	ID        uint      `dba:"title=群组ID;pk;incr"`
	Name      string    `dba:"title=群组名称;required"`
	CreatedAt time.Time `dba:"title=创建时间"`
}

type Tag struct {
	ID   uint   `dba:"title=标签ID;pk;incr"`
	Name string `dba:"title=标签名称;required"`
}

type UserGroup struct {
	*Group   `dba:"title=关联群组;rel=REF_MANY_SCHEMA"`
	GroupID  uint      `dba:"title=关联群组ID"`
	UserID   uint      `dba:"title=用户ID"`
	JoinedAt time.Time `dba:"title=加入时间"`
}
