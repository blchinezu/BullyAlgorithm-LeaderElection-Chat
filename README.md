# Bully Algorithm - Leader Election - Chat

### A GO implementation of the [Bully algorithm](https://en.wikipedia.org/wiki/Bully_algorithm) for [Joe Hudson](https://github.com/jhudson8)'s [chat example](https://github.com/jhudson8/golang-chat-example)

This [bully implementation](https://github.com/blchinezu/BullyAlgorithm-LeaderElection-Chat/blob/master/bully/bully.go) is meant to be used as a module with callback functions. The actual usage of the module is in [serverBully.go](https://github.com/blchinezu/BullyAlgorithm-LeaderElection-Chat/blob/master/serverBully.go). As you'll see, it's very easy to use it.

The server state is not maintained when switching between leaders so the chat room is not remembered and stuff like that. For that, there should've been implemented a distributed fault tolerant database too.

![screenshot.jpg](https://raw.githubusercontent.com/blchinezu/BullyAlgorithm-LeaderElection-Chat/master/screenshot.png)
