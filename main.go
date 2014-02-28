package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/alext/graceful_listener"
	"github.com/codegangsta/martini"
)

var (
	pubAddr  = getenvDefault("LINK_TRACKER_PUBADDR", ":8080")
	apiAddr  = getenvDefault("LINK_TRACKER_APIADDR", ":8081")
	inParent = os.Getenv("TEMPORARY_CHILD") != "1"
)

func getenvDefault(key string, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		val = defaultVal
	}

	return val
}

func main() {
	listener, apiListener, err := createListeners()
	if err != nil {
		log.Fatal(err)
	}

	reExec := make(chan bool)
	go handleSignal(listener, apiListener, reExec)

	m := martini.Classic()
	m.Get("/g", ExternalLinkTrackerHandler)
	mApi := martini.Classic()
	mApi.Put("/url", AddExternalURL)

	var wg sync.WaitGroup
	wg.Add(2)

	go serve(listener, m, &wg)
	log.Println("external-link-tracker: listening for redirects on " + listener.Addr().String())

	go serve(apiListener, mApi, &wg)
	log.Println("external-link-tracker: listening for writes on " + apiListener.Addr().String())

	if inParent {
		go stopTemporaryChild()
	}

	// Wait for serving routines to complete
	wg.Wait()

	if inParent {
		reExec <- true
		wg.Add(1)
		wg.Wait() // Wait forever...
	}

	os.Exit(0)
}

func createListeners() (listener, apiListener *graceful_listener.Listener, err error) {
	listenFD, err := strconv.Atoi(getenvDefault("LISTEN_FD", "0"))
	if err != nil {
		log.Println("Non-integer LISTEN_FD, ignoring:", err)
		listenFD = 0
	}
	apiListenFD, err := strconv.Atoi(getenvDefault("API_LISTEN_FD", "0"))
	if err != nil {
		log.Println("Non-integer API_LISTEN_FD, ignoring:", err)
		apiListenFD = 0
	}

	listener, err = graceful_listener.ResumeOrStart(listenFD, pubAddr)
	if err != nil {
		return nil, nil, err
	}
	apiListener, err = graceful_listener.ResumeOrStart(apiListenFD, apiAddr)
	if err != nil {
		return nil, nil, err
	}

	return
}

func serve(l *graceful_listener.Listener, handler http.Handler, wg *sync.WaitGroup) {
	err := http.Serve(l, handler)

	if l.Stopping() {
		err = l.WaitForClients(10)
		if err != nil {
			log.Println("all clients not closed", err)
		}
	} else if err != nil {
		log.Fatal("serve error: ", err)
	}
	wg.Done()
}

func handleSignal(l, lApi *graceful_listener.Listener, reExec chan bool) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	_ = <-c

	if inParent {
		upgradeServer(l, lApi, reExec)
	}
	l.Close()
	lApi.Close()
}

func upgradeServer(l, lApi *graceful_listener.Listener, reExec chan bool) {

	childPid, err := startTemporaryChild(l, lApi)
	if err != nil {
		log.Fatal(err)
	}

	fd, err := l.PrepareFd()
	if err != nil {
		log.Fatal(err)
	}
	fdApi, err := lApi.PrepareFd()
	if err != nil {
		log.Fatal(err)
	}

	go waitReExecSelf(fd, fdApi, childPid, reExec)
}

func waitReExecSelf(fd, fdApi, childPid int, reExec chan bool) {
	<-reExec // Wait until we're signalled to re-exec

	em := newEnvMap(os.Environ())
	em["LISTEN_FD"] = strconv.Itoa(fd)
	em["API_LISTEN_FD"] = strconv.Itoa(fdApi)
	em["TEMPORARY_CHILD_PID"] = strconv.Itoa(childPid)

	syscall.Exec(os.Args[0], os.Args, em.ToEnv())
}

func startTemporaryChild(l, lApi *graceful_listener.Listener) (pid int, err error) {
	fd, err := l.PrepareFd()
	if err != nil {
		log.Fatal(err)
	}
	fdApi, err := lApi.PrepareFd()
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(os.Args[0])
	em := newEnvMap(os.Environ())
	em["LISTEN_FD"] = strconv.Itoa(fd)
	em["API_LISTEN_FD"] = strconv.Itoa(fdApi)
	em["TEMPORARY_CHILD"] = "1"
	cmd.Env = em.ToEnv()

	log.Print("forking new server with cmd: ", cmd.Args)
	err = cmd.Start()
	if err != nil {
		return 0, err
	}
	return cmd.Process.Pid, nil
}

func stopTemporaryChild() {
	childPid, err := strconv.Atoi(getenvDefault("TEMPORARY_CHILD_PID", "0"))
	if err != nil {
		log.Println("non-integer in TEMPORARY_CHILD_PID, ignoring:", err)
		return
	}
	if childPid == 0 {
		// Nothing to do
		return
	}

	proc, err := os.FindProcess(childPid)
	if err != nil {
		log.Printf("Couldn't find child process with pid %d: %v", childPid, err)
		return
	}
	log.Println("Signalling child")
	err = proc.Signal(syscall.SIGHUP)
	if err != nil {
		log.Println("Error signalling child:", err)
	}
	log.Println("Waiting for child to exit")
	state, err := proc.Wait()
	if err != nil {
		log.Println("Error in Wait()", err)
		return
	}
	log.Printf("Child exited with status: %v", state)
}

type envMap map[string]string

func newEnvMap(env []string) (em envMap) {
	em = make(map[string]string, len(env))
	for _, item := range env {
		parts := strings.SplitN(item, "=", 2)
		em[parts[0]] = parts[1]
	}
	return
}

func (em envMap) ToEnv() (env []string) {
	env = make([]string, 0, len(em))
	for k, v := range em {
		env = append(env, k+"="+v)
	}
	return
}
