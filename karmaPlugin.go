package main

import (
	"database/sql"
	"log"
	"strconv"
)

type KarmaPlugin struct {
	Name string
}

type Karma struct {
	Id     int    `db:"id, primarykey, autoincrement"`
	User   string `db:"user, size:500"`
	Points int    `db:"points"`
}

func (kp KarmaPlugin) Register() (err error) {
	kp.Name = "Karma"
	return nil
}

func (kp KarmaPlugin) Parse(sender, channel, input string, conn *Connection) (err error) {
	if !Match(input, `\S+(\+|-){2,}`) {
		return nil
	}

	if channel == sender {
		conn.SendTo(sender, "Karma can only be modified in a public channel.")
		return nil
	}

	change := 0
	var user string
	user = MatchAndPull(input, `\S+\+\+`, `(\S+)\+\+`)
	if user != "" {
		change = 1
	} else {
		user = MatchAndPull(input, `\S+\-\-`, `(\S+)\-\-`)
		if user != "" {
			change = -1
		}
	}
	if change != 0 {
		if user == sender {
			conn.SendTo(channel, "I will not allow you to modify your own karma "+sender+".")
			return nil
		}
		var k Karma
		k, err = FindOrCreateKarma(user)
		if err != nil {
			log.Println("ERROR: Unable to find or create karma entry:", err)
			return
		}
		k.Points = k.Points + change
		err = k.Update()
		if err != nil {
			log.Println("ERROR: Unable to update karma entry:", err)
			return
		}
		conn.SendTo(channel, user+" now has "+strconv.Itoa(k.Points)+" karma.")
	}
	return nil
}

func (kp KarmaPlugin) Help() (helpText string) {
	return "<name>++ or <name>--"
}

func FindOrCreateKarma(u string) (k Karma, err error) {
	err = Db.SelectOne(&k, "select * from karma where user=?", u)
	if err != nil {
		if err == sql.ErrNoRows {
			k.Points = 0
			err = Db.Insert(&k)
			if err != nil {
				log.Println("ERROR: Unable to create new karma entry in db:", err)
				return
			}
		} else {
			log.Println("ERROR: Unable to select karma entry from db:", err)
		}
	}
	return
}

func (k *Karma) Update() (err error) {
	var rowCnt int64
	rowCnt, err = Db.Update(k)
	if err != nil {
		return err
	}
	if rowCnt == 0 {
		return ErrNoRowsUpdated
	}
	return nil
}
