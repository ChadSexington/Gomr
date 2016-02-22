package main

import (
	"bufio"
	"log"
	"regexp"
)

// All plugins should implement this interface
type Plugin interface {
	Register() error
	Parse(string, string, string, *Connection) error // Parse(user, channel, msg, connection)
	Help() string
}

var (
	database   Database
	config     Config
	pluginList []Plugin
)

// This is where we start heh
func main() {
	log.Println("Starting irc bot...")

	// Read configuration
	configPath := "./gomr.yaml"
	config = GetConfiguration(configPath)

	log.Println("Getting database connection...")
	database, err := InitDB(config.Db.Hostname, config.Db.Port,
		config.Db.Username, config.Db.Password, config.Db.Name)
	if err != nil {
		log.Panicln("ERROR: Unable to connect to to database:", err)
	}

	// TODO remove this, only here so the damn thing builds
	log.Println("remove me", database)

	// Register all plugins
	log.Println("Registering plugins...")
	registerPlugins()

	// create a connection to the irc server and join channel
	var conn *Connection
	conn, err = NewConnection(config.Hostname, config.Port, config.Channel, config.Nick)
	if err != nil {
		log.Panicln("Unable to connect to", config.Hostname, ":", config.Port, err)
	}

	// Loop through the connection stream for the rest of forseeable time
	stream := bufio.NewReader(conn.Conn)
	for {
		line, err := stream.ReadString('\n')
		if err != nil {
			log.Println("Oh shit, an error occured: ")
			log.Println(err)
			return
		}
		parseLine(line, conn)
	}

	// Close the connection if we ever get here for some reason
	conn.Conn.Close()
}

// All plugins should be registered here.
func registerPlugins() {
	exPlugin := ExamplePlugin{}
	err := exPlugin.Register()
	if err != nil {
		log.Println("Unable to register example plugin, skipping plugin.")
	} else {
		pluginList = append(pluginList, exPlugin)
	}

	karmaPlugin := KarmaPlugin{}
	err = karmaPlugin.Register()
	if err != nil {
		log.Println("Unable to register karma plugin, skipping plugin.")
	} else {
		pluginList = append(pluginList, karmaPlugin)
	}
}

// Main method to parse lines sent from the server
// Loops through each plugin in pluginList and runs the Parse() method from each
//   on the provided line
func parseLine(line string, conn *Connection) {
	log.Printf(line)

	// If a PING is received from the server, respond to avoid being disconnected
	if Match(line, "PING :"+config.Hostname+"$") {
		respondToPing(line, conn)
	} else {
		var user, channel, msg string
		var urgx, crgx, mrgx *regexp.Regexp

		// Example lines from server:
		// 2016/02/22 13:37:58 :tim!~tim@dhcp137-210.rdu.redhat.com PRIVMSG #test11123 :This is a test string
		// 2016/02/22 13:38:11 :tim!~tim@dhcp137-210.rdu.redhat.com NICK :timbo
		// 2016/02/22 13:38:13 :timbo!~tim@dhcp137-210.rdu.redhat.com PRIVMSG #test11123 :this is another test string
		urgx = regexp.MustCompile(`:(\S+)!~`)
		umatch := urgx.FindStringSubmatch(line)
		if umatch != nil && len(umatch) > 1 {
			user = umatch[1]
			log.Println("user:", user)
		}

		crgx = regexp.MustCompile(`\sPRIVMSG\s(\S+)\s`)
		cmatch := crgx.FindStringSubmatch(line)
		if cmatch != nil && len(cmatch) > 1 {
			channel = cmatch[1]
			log.Println("channel:", channel)
		}

		mrgx = regexp.MustCompile(`\sPRIVMSG\s\S+\s:(.*)`)
		mmatch := mrgx.FindStringSubmatch(line)
		if mmatch != nil && len(mmatch) > 1 {
			msg = mmatch[1]
			log.Println("message:", msg)
		}

		if msg != "" {
			for _, plugin := range pluginList {
				plugin.Parse(user, channel, msg, conn)
			}
		}
	}
}

// Respond to pings from the irc server to keep the server alive
func respondToPing(ping string, conn *Connection) {
	conn.Send("PONG " + config.Hostname)
	log.Println("PONG " + config.Hostname)
}
