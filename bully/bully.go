package bully

import (
    "bufio"
    "fmt"
    "net"
    "os"
    "strconv"
    "time"
)

// =============================================================================
// ================================= PRIVATE ===================================
// =============================================================================

var verboseMessages bool = false

var pingInterval time.Duration = 1 * time.Second

var responseTimeout time.Duration = 1 * time.Second

var callbackOnBecomingLeader func()
var callbackOnLosingLeadership func()

type bully_s struct {
    id     uint64
    addr   string
    port   string
    active bool
    leader bool
}

// Self
var self bully_s = bully_s{
    id:     6660,
    addr:   "127.0.0.1",
    port:   "6660",
    active: true,
    leader: false,
}

// Others
var bullies []bully_s

// State changers
func becomeLeader() {
    triggerCallback := !self.leader

    for i := 0; i < len(bullies); i++ {
        bullies[i].leader = false
    }
    self.leader = true

    if callbackOnBecomingLeader != nil && triggerCallback {
        go callbackOnBecomingLeader()
    }
}

func loseLeadershipTo(leaderId uint64) {
    triggerCallback := self.leader

    self.leader = false

    for i := 0; i < len(bullies); i++ {
        if bullies[i].id == leaderId {
            bullies[i].leader = true
            bullies[i].active = true
        } else {
            bullies[i].leader = false
        }
    }

    if callbackOnLosingLeadership != nil && triggerCallback {
        go callbackOnLosingLeadership()
    }
}

func launchListener() {

    verboseMessage("Launch bully listener...")
    ln, err := net.Listen("tcp", self.addr+":"+self.port)
    exitOnError(err)

    for {
        // verboseMessage("Wait for connections...")
        conn, err := ln.Accept()
        if err != nil {
            message("WARN:", err)
            continue
        }

        go handleNewMessage(conn)
    }
}

func handleNewMessage(conn net.Conn) {

    str, err := bufio.NewReader(conn).ReadString('\n')
    exitOnError(err)

    verboseMessage("Handle new client message: \"" + str[:len(str)-1] + "\"")

    if str[:8] == "ELECTION" {
        conn.Write([]byte("OK\n"))
        go announceElection(numberFromString(str[9 : len(str)-1]))
    } else if str[:6] == "LEADER" {
        conn.Write([]byte("OK\n"))
        go loseLeadershipTo(numberFromString(str[7 : len(str)-1]))
    } else if str[:4] == "PING" {
        conn.Write([]byte("OK\n"))
    } else {
        verboseMessage("Received invalid message: \"", str[:len(str)-1], "\"")
    }
    conn.Close()
}

func announceElection(announcerId uint64) {
    verboseMessage("Announce election from", announcerId)

    announcements := 0
    for i := 0; i < len(bullies); i++ {

        // Mark the sender as being active
        if bullies[i].id == announcerId {
            bullies[i].active = true
        }

        // Skip inactive
        if !bullies[i].active {
            continue
        }

        // Skip lower IDs
        if bullies[i].id <= self.id {
            continue
        }

        // Connect
        conn, err := net.Dial("tcp", bullies[i].addr+":"+bullies[i].port)
        if err != nil {
            bullies[i].active = false
            // verboseMessage(err)
            verboseMessage("Marked inactive B", bullies[i].id)
            continue
        }
        bullies[i].active = true

        // Send message
        _, err = fmt.Fprintf(conn, "ELECTION "+self.port+"\n")
        if err != nil {
            verboseMessage(err)
            verboseMessage("Couldn't announce ELECTION to B", bullies[i].id)
            continue
        }

        // Set timeout
        conn.SetReadDeadline(time.Now().Add(responseTimeout))

        // Wait for response
        str, err := bufio.NewReader(conn).ReadString('\n')
        if err != nil {
            verboseMessage(err)
            verboseMessage("Error receiving OK message from B", bullies[i].id)
            continue
        }

        // Mark
        if str == "OK\n" {
            announcements++
        } else {
            verboseMessage("Received unknown response from B", bullies[i].id, " \"", str[:len(str)-1], "\"")
        }

        // Close
        conn.Close()

        if announcements == 1 {
            bullies[i].leader = true
            break
        }
    }

    verboseMessage("Sent", announcements, "election announcements")

    if announcements == 0 {
        announceImLeader()
    }
}

func announceImLeader() {
    if self.leader {
        verboseMessage("I'm already leading!")
        return
    }

    verboseMessage("Announcing leadership")

    announcements := 0
    for i := 0; i < len(bullies); i++ {

        // Skip inactive
        if !bullies[i].active {
            continue
        }

        // Connect
        conn, err := net.Dial("tcp", bullies[i].addr+":"+bullies[i].port)
        if err != nil {
            bullies[i].active = false
            // verboseMessage(err)
            verboseMessage("Marked inactive B", bullies[i].id)
            continue
        }
        bullies[i].active = true

        // Send message
        _, err = fmt.Fprintf(conn, "LEADER "+self.port+"\n")
        if err != nil {
            verboseMessage(err)
            verboseMessage("Couldn't announce LEADER to B", bullies[i].id)
            continue
        }

        // Set timeout
        conn.SetReadDeadline(time.Now().Add(responseTimeout))

        // Wait for response
        str, err := bufio.NewReader(conn).ReadString('\n')
        if err != nil {
            verboseMessage(err)
            verboseMessage("Error receiving OK message from B", bullies[i].id)
            continue
        }

        // Mark
        if str == "OK\n" {
            announcements++
        } else {
            verboseMessage("Received unknown response from B", bullies[i].id, " \"", str[:len(str)-1], "\"")
        }

        // Close
        conn.Close()

        // Mark
        announcements++
    }

    // Log
    verboseMessage("Sent", announcements, "leader announcements")

    // Become leader
    becomeLeader()
}

func monitorLeader() {
    for {

        time.Sleep(pingInterval)

        if self.leader {
            continue
        }

        verboseMessage("Ping leader")

        leaders := 0
        pings := 0
        for i := 0; i < len(bullies); i++ {

            // Skip nonleading punks
            if !bullies[i].leader {
                continue
            }

            // Count leaders
            leaders++

            // Only ping the first leader
            if leaders > 1 {
                continue
            }

            // Connect
            conn, err := net.Dial("tcp", bullies[i].addr+":"+bullies[i].port)
            if err != nil {
                bullies[i].active = false
                verboseMessage(err)
                verboseMessage("Couldn't connnect to B", bullies[i].id)
                continue
            }

            // Send message
            _, err = fmt.Fprintf(conn, "PING "+self.port+"\n")
            if err != nil {
                verboseMessage(err)
                verboseMessage("Couldn't ping LEADER (B", bullies[i].id, ")")
                continue
            }

            // Set timeout
            conn.SetReadDeadline(time.Now().Add(responseTimeout))

            // Wait for response
            str, err := bufio.NewReader(conn).ReadString('\n')
            if err != nil {
                verboseMessage(err)
                verboseMessage("Error receiving OK message while pinging B", bullies[i].id)
                continue
            }

            // Mark
            if str == "OK\n" {
                verboseMessage("Got leader:", bullies[i].id)
                pings++
            } else {
                verboseMessage("Received unknown response from leader B", bullies[i].id, " \"", str[:len(str)-1], "\"")
            }

            // Close
            conn.Close()
        }

        verboseMessage(leaders, "leaders,", pings, "pings")

        // Start election if there's no leader or there are more than 1
        // Or there are unsuccessful pings
        if leaders != 1 || pings != leaders {
            announceElection(0)
        }
    }
}

func checkActiveBullies() {
    for i := 0; i < len(bullies); i++ {

        // Connect
        conn, err := net.Dial("tcp", bullies[i].addr+":"+bullies[i].port)
        if err != nil {
            continue
        }

        // Send message
        _, err = fmt.Fprintf(conn, "PING "+self.port+"\n")
        if err != nil {
            continue
        }

        // Set timeout
        conn.SetReadDeadline(time.Now().Add(responseTimeout))

        // Wait for response
        str, err := bufio.NewReader(conn).ReadString('\n')
        if err != nil {
            continue
        }

        // Mark
        if str == "OK\n" {
            bullies[i].active = true
            message("Mark active B", bullies[i].id)
        }

        // Close
        conn.Close()
    }
}

// =============================================================================
// ================================= PRIVATE ===================================
// =============================================================================

func message(a ...interface{}) (n int, err error) {
    return fmt.Print("[B", self.port, "] ", fmt.Sprintln(a...))
}

func exitOnError(err error) {
    if err != nil {
        message("ERR:", err)
        os.Exit(1)
    }
}

func verboseMessage(a ...interface{}) {
    if verboseMessages {
        message(a...)
    }
}

func thisIsMe(bully bully_s) bool {
    if bully.addr == "127.0.0.1" || bully.addr == "localhost" {
        if bully.port == self.port {
            return true
        }
    }
    return false
}

func buildNewBully(str string) bully_s {

    host, port, err := net.SplitHostPort(str)
    exitOnError(err)

    bully := bully_s{
        id:     numberFromString(port),
        addr:   host,
        port:   port,
        active: false,
        leader: false,
    }

    return bully
}

func numberFromString(str string) uint64 {
    number, err := strconv.ParseUint(str, 10, 64)
    exitOnError(err)
    return number
}

// =============================================================================
// ================================== PUBLIC ===================================
// =============================================================================

func SetCallbackOnBecomingLeader(callback func()) {
    callbackOnBecomingLeader = callback
}

func SetCallbackOnLosingLeadership(callback func()) {
    callbackOnLosingLeadership = callback
}

func SetSelf(str string) {
    newBully := buildNewBully(str)
    newBully.active = true
    self = newBully
}

func SetBullies(bullies_list []string) {
    for _, str := range bullies_list {
        newBully := buildNewBully(str)

        if !thisIsMe(newBully) {
            bullies = append(bullies, newBully)
        }
    }
}

func StartBully(lockExecution bool) {

    if len(bullies) < 1 {
        fmt.Println("ERROR: There are no bullies to run on!")
        return
    }

    go launchListener()
    checkActiveBullies()
    if lockExecution {
        monitorLeader()
    } else {
        go monitorLeader()
    }
}
