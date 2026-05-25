package main

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type User struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var u = []User{
	{
		Id:    1,
		Name:  "John Doe",
		Email: "john.doe@example.com",
	},
	{
		Id:    2,
		Name:  "Jane Smith",
		Email: "jane.smith@example.com",
	},
	{
		Id:    3,
		Name:  "Bob Johnson",
		Email: "bob.johnson@example.com",
	},
}

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "My Go App!",
		})
	})

	r.GET("/users", func(c *gin.Context) {
		c.JSON(200, u)
	})

	r.GET("/users/:id", func(c *gin.Context) {
		idParam := c.Param("id")
		id, err := strconv.Atoi(idParam)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}
		for _, user := range u {
			if id == user.Id {
				c.JSON(200, user)
				return
			}
		}
		c.JSON(404, gin.H{"error": "User not found"})
	})

	r.POST("/users", func(c *gin.Context) {
		var user User
		if err := c.BindJSON(&user); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		user.Id = len(u) + 1
		u = append(u, user)
		c.JSON(201, user)
	})

	r.PUT("/users/:id", func(c *gin.Context) {
		idParam := c.Param("id")
		id, err := strconv.Atoi(idParam)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		var user User
		if err := c.BindJSON(&user); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		for i, existingUser := range u {
			if id == existingUser.Id {
				user.Id = existingUser.Id
				u[i] = user
				c.JSON(200, user)
				return
			}
		}
		c.JSON(404, gin.H{"error": "User not found"})
	})

	r.DELETE("/users/:id", func(c *gin.Context) {
		idParam := c.Param("id")
		id, err := strconv.Atoi(idParam)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		for i, user := range u {
			if id == user.Id {
				u = append(u[:i], u[i+1:]...)
				c.JSON(204, nil)
				return
			}
		}
		c.JSON(404, gin.H{"error": "User not found"})
	})

	r.Run()
}
