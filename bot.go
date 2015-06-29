package main

import (
  "os"
  "os/user"
  "flag"
  "fmt"
  "log"
  "path"
  "regexp"
  "strings"

  "github.com/nickvanw/ircx"
  "github.com/sorcix/irc"
  "github.com/luksen/maildir"
)

var (
  name = flag.String("name", "testgoircx", "Nick to use in IRC")
  server = flag.String("server", "chat.freenode.org:6667", "Host:port to connect to")
  channels = flag.String("chan", "#botwar", "Channels")
)

func init() {
  flag.Parse()
}

func main() {
  bot := ircx.Classic(*server, *name)
  if err := bot.Connect(); err != nil {
    log.Panicln("Unable to dial IRC server ", err)
  }
  RegisterHandlers(bot)
  bot.CallbackLoop()
  log.Println("Exiting...")
}

func RegisterHandlers(bot *ircx.Bot) {
  bot.AddCallback(irc.RPL_WELCOME, ircx.Callback{Handler: ircx.HandlerFunc(RegisterConnect)})
  bot.AddCallback(irc.PING, ircx.Callback{Handler: ircx.HandlerFunc(PingHandler)})

  maildirproxy := NewMaildirproxy(*server)
  bot.AddCallback(irc.PRIVMSG, ircx.Callback{Handler: ircx.HandlerFunc(maildirproxy.PrivmsgHandler)})
}

func RegisterConnect(s ircx.Sender, m *irc.Message) {
  s.Send(&irc.Message{
    Command: irc.JOIN,
    Params: []string{*channels},
  })
}

func PingHandler(s ircx.Sender, m *irc.Message) {
  s.Send(&irc.Message{
    Command: irc.PONG,
    Params: m.Params,
    Trailing: m.Trailing,
  })
}

type Maildirproxy struct {
  Server string
  Mdir maildir.Dir
}

func Formatservername(name string) string {
  host := strings.Split(name, ":")
  reg, err := regexp.Compile("[^A-Za-z0-9]+")
  if err != nil {
    log.Fatal(err)
  }

  safe := reg.ReplaceAllString(host[0], "")
  safe = "." + strings.ToLower(strings.Trim(safe, " "))

  usr, err := user.Current()
  if err != nil {
    log.Fatal(err)
  }

  full := path.Join(usr.HomeDir, safe)
  return full
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	panic(err)
}

func NewMaildirproxy(srv string) *Maildirproxy {
  normname := Formatservername(srv)
  var d maildir.Dir = maildir.Dir(normname)

  if !exists(normname) {
    fmt.Printf("creating maildir (%s) \n", normname)
    err := d.Create()
    if err != nil {
      log.Fatal(err)
    }
  }

  m := &Maildirproxy {
    Server: srv,
    Mdir: d,
  }
  return m
}

func (p *Maildirproxy) PrivmsgHandler(s ircx.Sender, m *irc.Message) {
  cmd := m.Command
  trl := m.Trailing
  fmt.Printf("%s (%s) @ %s / ", m.Prefix.Name, m.Prefix.User, m.Prefix.Host)
  fmt.Printf("%s / ", cmd)
  fmt.Printf("%s ", m.Params[0])
  fmt.Printf("%s mdir-%s\n", trl, p.Server)

  msg := "From: outsider <"+ m.Prefix.User + "@" + m.Prefix.Host + ">\nTo: local dude <dude@localhost>" + "\nSubject: " + m.Params[0] + "\nDate: Fri, 21 Nov 1997 09:55:06 -0600" + "\nMessage-ID: <1234@localhost>"+ "\n\n" + trl + "\n"
  log.Println("debug>>" + msg)

  dlv, err := p.Mdir.NewDelivery()
  if err != nil {
    log.Fatal(err)
  }

  _, err = dlv.Write([]byte(msg))
  if err != nil {
    log.Fatal(err)
  }

  err = dlv.Close()
  if err != nil {
    log.Fatal(err)
  }

}
