package main

import (
    "./bully"
    "./util"
    "fmt"
    "os"
    "os/exec"
    "strings"
)

var cmd *exec.Cmd

// Launch server when becoming leader
func becameLeader() {
    fmt.Println("Server " + os.Args[1] + ": Became Leader")

    // Build child command
    cmd = exec.Command("./server")

    // Launch command
    err := cmd.Start()
    if err != nil {
        fmt.Println("ERR:", err)
        os.Exit(1)
    }
    err = cmd.Wait()
}

// Kill server when losing leadership
func lostLeadership() {
    fmt.Println("Server " + os.Args[1] + ": Lost Leadership")

    // cmd.Process.Kill()
    if err := cmd.Process.Kill(); err != nil {
        fmt.Println("ERR:", err)
        os.Exit(1)
    }
}

// main program
func main() {

    // Get config
    properties := util.LoadConfig()

    // Set bully config
    // set from cmd arg for easier testing purpose
    bully.SetSelf("127.0.0.1:666" + os.Args[1])
    // bully.SetSelf(properties.SelfBully)
    bully.SetBullies(strings.Split(properties.AllBullies, ","))
    bully.SetCallbackOnBecomingLeader(becameLeader)
    bully.SetCallbackOnLosingLeadership(lostLeadership)

    // Launch bully service
    bully.StartBully(true)
}
