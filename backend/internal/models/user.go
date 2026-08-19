package models

import "gorm.io/gorm"

type User struct {
	gorm.Model

	Name     string `gorm:"size:100;not null"`
	Username string `gorm:"size:50;uniqueIndex;not null"`
	Email    string `gorm:"size:150;uniqueIndex"`
	Password string `gorm:"not null"`
	Role     string `gorm:"default:user"`
	Active   bool   `gorm:"default:true"`
	AgentID  *uint  `gorm:"index"`
	Agent    *Agent `gorm:"foreignKey:AgentID"`

	CustomerID *uint     `gorm:"uniqueIndex"`
	Customer   *Customer `gorm:"foreignKey:CustomerID"`
}
