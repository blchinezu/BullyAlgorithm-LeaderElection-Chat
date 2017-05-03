package main

import (
    "./util"
    "bufio"
    "fmt"
    "net"
    "os"
    "regexp"
    "strings"
    "time"
)

// input message regular expression (look for a command /whatever)
var standardInputMessageRegex, _ = regexp.Compile(`^\/([^\s]*)\s*(.*)$`)

// chat server command /command [username] body contents
var chatServerResponseRegex, _ = regexp.Compile(`^\/([^\s]*)\s?(?:\[([^\]]*)\])?\s*(.*)$`)

// container for chat server Command details
type Command struct {
    // "leave", "message", "enter"
    Command, Username, Body string
}

var conn net.Conn

var quit bool = false
var connectionNb uint64 = 0

func reloadClient(showReconnectingOutput bool) {

    if quit {
        return
    }

    // keep it closed range
    if connectionNb > 100 {
        connectionNb = 0
    } else {
        connectionNb++
    }

    // get config
    username, properties := getConfig()

    // connect
    if showReconnectingOutput {
        fmt.Println("Reconnecting...")
    } else {
        fmt.Println("Connecting...")
    }
    var err error
    for {
        conn, err = net.Dial("tcp", properties.Hostname+":"+properties.Port)
        if err != nil {
            time.Sleep(time.Second)
        } else {
            break
        }
    }

    // fmt.Println("Connected")
    // we're listening to chat server commands *and* user terminal commands
    go watchForConnectionInput(username, properties, connectionNb)
    go watchForConsoleInput(connectionNb)
}

// program main
func main() {
    go reloadClient(false)
    for {
        if quit {
            break
        }
        time.Sleep(100 * time.Millisecond)
    }
}

// parse out the arguments to be used when connecting to the chat server
func getConfig() (string, util.Properties) {
    if len(os.Args) >= 2 {
        username := os.Args[1]
        properties := util.LoadConfig()
        return username, properties
    } else {
        println("You must provide the username as the first parameter ")
        os.Exit(1)
        return "", util.Properties{}
    }
}

// keep watching for console input
// send the "message" command to the chat server when we have some
func watchForConsoleInput(nb uint64) {
    reader := bufio.NewReader(os.Stdin)

    for true {

        if nb != connectionNb {
            return
        }

        message, err := reader.ReadString('\n')
        if err != nil {
            // fmt.Println("Console: ", err)
            // util.CheckForError(err, "Lost console connection")
            return
        }

        message = strings.TrimSpace(message)
        if message != "" {
            command := parseInput(message)

            if command.Command == "" {
                // there is no command so treat this as a simple message to be sent out
                sendCommand("message", message, conn)
            } else {
                switch command.Command {

                // enter a room
                case "enter":
                    sendCommand("enter", command.Body, conn)

                // ignore someone
                case "ignore":
                    sendCommand("ignore", command.Body, conn)

                // leave a room
                case "leave":
                    // leave the current room (we aren't allowing multiple rooms)
                    sendCommand("leave", "", conn)

                // disconnect from the chat server
                case "disconnect":
                    sendCommand("disconnect", "", conn)
                    quit = true

                default:
                    fmt.Printf("Unknown command \"%s\"\n", command.Command)
                }
            }
        }
    }
}

// listen for any commands that come from the chat server
// like someone entered the room, said something, or left the room
func watchForConnectionInput(username string, properties util.Properties, nb uint64) {
    reader := bufio.NewReader(conn)

    for true {

        if nb != connectionNb {
            return
        }

        message, err := reader.ReadString('\n')
        if err != nil {
            // fmt.Println("Connection: ", err)
            // util.CheckForError(err, "Lost server connection")
            go reloadClient(true)
            return
        }
        message = strings.TrimSpace(message)
        if message != "" {
            Command := parseCommand(message)
            switch Command.Command {

            // the handshake - send out our username
            case "ready":
                sendCommand("user", username, conn)

            // the user has connected to the chat server
            case "connect":
                fmt.Printf(properties.HasEnteredTheLobbyMessage+"\n", Command.Username)

            // the user has disconnected
            case "disconnect":
                fmt.Printf(properties.HasLeftTheLobbyMessage+"\n", Command.Username)

            // the user has entered a room
            case "enter":
                fmt.Printf(properties.HasEnteredTheRoomMessage+"\n", Command.Username, Command.Body)

            // the user has left a room
            case "leave":
                fmt.Printf(properties.HasLeftTheRoomMessage+"\n", Command.Username, Command.Body)

            // the user has sent a message
            case "message":
                if Command.Username != username {
                    fmt.Printf(properties.ReceivedAMessage+"\n", Command.Username, Command.Body)
                }

            // the user has connected to the chat server
            case "ignoring":
                fmt.Printf(properties.IgnoringMessage+"\n", Command.Body)
            }
        }
    }
}

// send a command to the chat server
// commands are in the form of /command {command specific body content}\n
func sendCommand(command string, body string, conn net.Conn) {
    message := fmt.Sprintf("/%v %v\n", util.Encode(command), util.Encode(body))
    conn.Write([]byte(message))
}

// parse the input message and return an Command
// if there is a command the "Command" will != "", otherwise just Body will exist
func parseInput(message string) Command {
    res := standardInputMessageRegex.FindAllStringSubmatch(message, -1)
    if len(res) == 1 {
        // there is a command
        return Command{
            Command: res[0][1],
            Body:    res[0][2],
        }
    } else {
        return Command{
            Body: util.Decode(message),
        }
    }
}

// look for "/Command [name] body contents" where [name] is optional
func parseCommand(message string) Command {
    res := chatServerResponseRegex.FindAllStringSubmatch(message, -1)
    if len(res) == 1 {
        // we've got a match
        return Command{
            Command:  util.Decode(res[0][1]),
            Username: util.Decode(res[0][2]),
            Body:     util.Decode(res[0][3]),
        }
    } else {
        // it's irritating that I can't return a nil value here - must be something I'm missing
        return Command{}
    }
}
