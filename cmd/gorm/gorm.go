package main

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
)

func main() {
	// https://github.com/go-sql-driver/mysql
	dsn := "root:wosoft2019@tcp(code.mtm.ink:8013)/ai_ocr?tls=skip-verify&charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai"
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	sql := "CREATE TABLE IF NOT EXISTS `account` (\n`user_id`\tBIGINT UNSIGNED NOT NULL COMMENT '关联用户ID',\n`type`\tTEXT NOT NULL COMMENT '账户类型',\n`username`\tTEXT NOT NULL COMMENT '账户名称',\n`password`\tTEXT NOT NULL COMMENT '账户密码',\n`remark`\ttext NOT NULL COMMENT '备注',\n`id`\tBIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\nPRIMARY KEY (`id`)\n);\n\nCREATE TABLE IF NOT EXISTS `org` (\n`deleted_at`\tDATETIME(3) NULL COMMENT '逻辑删除时间',\n`remark`\ttext NULL COMMENT '备注',\n`id`\tBIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n`created_at`\tDATETIME(3) NOT NULL COMMENT '创建时间',\n`code`\tTEXT NOT NULL,\n`name`\tTEXT NOT NULL,\n`updated_at`\tDATETIME(3) NOT NULL COMMENT '最后修改时间',\nPRIMARY KEY (`id`)\n);\n\nCREATE TABLE IF NOT EXISTS `permission` (\n`id`\tBIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n`code`\tTEXT NOT NULL COMMENT '权限代码',\n`name`\tTEXT NOT NULL COMMENT '权限名称',\nPRIMARY KEY (`id`)\n);\n\nCREATE TABLE IF NOT EXISTS `profile` (\n`id`\tBIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n`user_id`\tBIGINT UNSIGNED NOT NULL COMMENT '关联用户ID',\n`real_name`\tTEXT NOT NULL COMMENT '真实姓名',\nPRIMARY KEY (`id`)\n);\n\nCREATE TABLE IF NOT EXISTS `role` (\n`id`\tBIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n`created_at`\tDATETIME(3) NOT NULL COMMENT '创建时间',\n`updated_at`\tDATETIME(3) NOT NULL COMMENT '最后修改时间',\n`deleted_at`\tDATETIME(3) NULL COMMENT '逻辑删除时间',\n`remark`\ttext NULL COMMENT '备注',\n`name`\tTEXT NOT NULL COMMENT '角色名称',\nPRIMARY KEY (`id`)\n);\n\nCREATE TABLE IF NOT EXISTS `user` (\n`id`\tBIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n`nickname`\tTEXT NOT NULL COMMENT '用户昵称',\n`org_id`\tBIGINT UNSIGNED NOT NULL COMMENT '所属组织ID',\nPRIMARY KEY (`id`)\n);\n\nCREATE TABLE IF NOT EXISTS `user_dept` (\n`dept_id`\tBIGINT UNSIGNED NOT NULL COMMENT '关联部门ID',\n`is_main`\tTINYINT(1) NULL COMMENT '是否主部门',\n`user_id`\tBIGINT UNSIGNED NOT NULL COMMENT '关联用户ID',\nPRIMARY KEY (`dept_id`,`user_id`)\n);"

	if res := db.Exec(sql); res.Error != nil {
		log.Fatalln(res.Error)
	} else {
		log.Println(res.RowsAffected)
	}
}
