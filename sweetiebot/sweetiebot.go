package sweetiebot

import (
  "fmt"
  "strconv"
  "time"
  "io/ioutil"
  "github.com/bwmarrin/discordgo"
  "database/sql"
  "strings"
  "encoding/json"
  "reflect"
  "math/rand"
  "regexp"
  "encoding/base64"
)

type ModuleHooks struct {
    OnEvent                   []ModuleOnEvent
    OnTypingStart             []ModuleOnTypingStart
    OnMessageCreate           []ModuleOnMessageCreate
    OnMessageUpdate           []ModuleOnMessageUpdate
    OnMessageDelete           []ModuleOnMessageDelete
    OnMessageAck              []ModuleOnMessageAck
    OnUserUpdate              []ModuleOnUserUpdate
    OnPresenceUpdate          []ModuleOnPresenceUpdate
    OnVoiceStateUpdate        []ModuleOnVoiceStateUpdate
    OnGuildUpdate             []ModuleOnGuildUpdate
    OnGuildMemberAdd          []ModuleOnGuildMemberAdd
    OnGuildMemberRemove       []ModuleOnGuildMemberRemove
    OnGuildMemberUpdate       []ModuleOnGuildMemberUpdate
    OnGuildBanAdd             []ModuleOnGuildBanAdd
    OnGuildBanRemove          []ModuleOnGuildBanRemove
    OnCommand                 []ModuleOnCommand
    OnIdle                    []ModuleOnIdle
}

type BotConfig struct {
  Debug bool               `json:"debug"`
  Maxerror int64           `json:"maxerror"`
  Maxwit int64             `json:"maxwit"`
  Maxspam int              `json:"maxspam"`
  Maxbored int64           `json:"maxbored"`
  Maxspoiltime int64       `json:"maxspoiltime"`       
  MaxPMlines int           `json:"maxpmlines"`
  Maxquotelines int        `json:"maxquotelines"`
  Maxmarkovlines int       `json:"maxmarkovlines"`
  Maxsearchresults int     `json:"maxsearchresults"`
  Defaultmarkovlines int   `json:"defaultmarkovlines"`
  Maxshutup int64          `json:"maxshutup"`
  Maxcute int64            `json:"maxcute"`
  Commandperduration int   `json:"commandperduration"`
  Commandmaxduration int64 `json:"commandmaxduration"`
  StatusDelayTime int      `json:"statusdelaytime"`
  MaxRaidTime int64        `json:"maxraidtime"`
  RaidSize int             `json:"raidsize"`
  Witty map[string]string  `json:"witty"`
  Aliases map[string]string `json:"aliases"`
  MaxBucket int            `json:"maxbucket"`
  MaxBucketLength int      `json:"maxbucketlength"`
  MaxFightHP int           `json:"maxfighthp"`
  MaxFightDamage int       `json:"maxfightdamage"`
  AlertRole uint64         `json:"alertrole"`
  SilentRole uint64        `json:"silentrole"`
  LogChannel uint64        `json:"logchannel"`
  ModChannel uint64        `json:"modchannel"`
  SpoilChannels []uint64   `json:"spoilchannels"`
  FreeChannels map[string]bool    `json:"freechannels"`
  Command_roles map[string]map[string]bool    `json:"command_roles"`
  Command_channels map[string]map[string]bool  `json:"command_channels"`
  Command_limits map[string]int64 `json:command_limits`
  Command_disabled map[string]bool `json:command_disabled`
  Module_disabled map[string]bool `json:module_disabled`
  Module_channels map[string]map[string]bool `json:module_channels`
  Collections map[string]map[string]bool `json:"collections"`
  Groups map[string]map[string]bool `json:"groups"`
}

type GuildInfo struct {
  GuildOwner uint64
}

type SweetieBot struct {
  db *BotDB
  log *Log
  dg *discordgo.Session
  SelfID string
  OwnerID uint64
  DebugChannelID string
  version string
  hooks ModuleHooks
  quit bool
  emotemodule *EmoteModule
  guilds map[string]GuildInfo
  Guild *discordgo.Guild
  command_last map[string]map[string]int64
  commandlimit *SaturationLimit
  config BotConfig
  initialized bool
  modules []Module
  commands map[string]Command
}

var sb *SweetieBot
var channelregex = regexp.MustCompile("<#[0-9]+>")
var userregex = regexp.MustCompile("<@!?[0-9]+>")

func (sbot *SweetieBot) AddCommand(c Command) {
  sbot.commands[strings.ToLower(c.Name())] = c
}

func (sbot *SweetieBot) SaveConfig() {
  data, err := json.Marshal(sb.config)
  if err == nil {
    ioutil.WriteFile("config.json", data, 0)
  } else {
    sbot.log.Log("Error writing json: ", err.Error())
  }
}

func (sbot *SweetieBot) SetConfig(name string, value string, extra... string) (string, bool) {
  name = strings.ToLower(name)
  t := reflect.ValueOf(&sbot.config).Elem()
  n := t.NumField()
  for i := 0; i < n; i++ {
    if strings.ToLower(t.Type().Field(i).Name) == name {
      f := t.Field(i)
      switch t.Field(i).Interface().(type) {
        case string:
          f.SetString(value)
        case int, int8, int16, int32, int64:
          k, _ := strconv.ParseInt(value, 10, 64)
          f.SetInt(k)
        case uint, uint8, uint16, uint32:
          k, _ := strconv.ParseUint(value, 10, 64)
          f.SetUint(k)
        case uint64:
          f.SetUint(PingAtoi(value))
        case []uint64:
          f.Set(reflect.MakeSlice(reflect.TypeOf(f.Interface()), 0, 1 + len(extra)))
          f.Set(reflect.Append(f, reflect.ValueOf(PingAtoi(value))))
          for _, k := range extra {
            f.Set(reflect.Append(f, reflect.ValueOf(PingAtoi(k))))
          }
        case bool:
          f.SetBool(value == "true")
        case map[string]string:
          if len(extra) == 0 {
            sbot.log.Log("No extra parameter given for " + name)
            return "", false
          }
          if f.IsNil() {
            f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
          }
          f.SetMapIndex(reflect.ValueOf(value), reflect.ValueOf(extra[0]))
        case map[string]int64:
          if len(extra) == 0 {
            sbot.log.Log("No extra parameter given for " + name)
            return "", false
          }
          if f.IsNil() {
            f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
          }
          k, _ := strconv.ParseInt(extra[0], 10, 64)
          f.SetMapIndex(reflect.ValueOf(value), reflect.ValueOf(k))
        case map[string]bool:
          f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
          f.SetMapIndex(reflect.ValueOf(value), reflect.ValueOf(true))
          for _, k := range extra {
            f.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(true))
          }
        case map[string]map[string]bool:
          if len(extra) < 2 {
            sbot.log.Log("Not enough parameters for " + name)
            return "", false
          }
          if f.IsNil() {
            f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
          }
        default:
          sbot.log.Log(name + " is an unknown type " + t.Field(i).Type().Name())
          return "", false
      }
      sbot.SaveConfig()
      return fmt.Sprint(t.Field(i).Interface()), true
    }
  }
  return "", false
}

func sbemotereplace(s string) string {
  return strings.Replace(s, "[](/", "[\u200B](/", -1)
}

func SanitizeOutput(message string) string {
  if sb.emotemodule != nil {
    message = sb.emotemodule.emoteban.ReplaceAllStringFunc(message, sbemotereplace)
  }
  return message;
}
func ExtraSanitize(s string) string {
  s = strings.Replace(s,"`","",-1)
  s = strings.Replace(s, "[](/", "[\u200B](/", -1)
  s = strings.Replace(s, "http://", "http\u200B://", -1)
  s = strings.Replace(s, "https://", "https\u200B://", -1)
  return s
}

func (sbot *SweetieBot) SendMessage(channelID string, message string) {
  sbot.dg.ChannelMessageSend(channelID, SanitizeOutput(message));
}

func ProcessModule(channelID string, m Module) bool {
  _, disabled := sb.config.Module_disabled[m.Name()]
  if disabled { return false }
    
  c := sb.config.Module_channels[m.Name()]
  if len(channelID)>0 && len(c)>0 { // Only check for channels if we have a channel to check for, and the module actually has specific channels
    _, ok := c[channelID]
    return ok
  }
  return true
}

func SwapStatusLoop() {
  for !sb.quit {
    if len(sb.config.Collections["status"]) > 0 {
      sb.dg.UpdateStatus(0, MapGetRandomItem(sb.config.Collections["status"]))
    }
    time.Sleep(time.Duration(sb.config.StatusDelayTime)*time.Second)
  }
}

func ChangeBotName(s *discordgo.Session, name string, avatarfile string) {
  binary, _ := ioutil.ReadFile(avatarfile)
  email, _ := ioutil.ReadFile("email")
  password, _ := ioutil.ReadFile("passwd")
  avatar := base64.StdEncoding.EncodeToString(binary)
    
  _, err := s.UserUpdate(strings.TrimSpace(string(email)), strings.TrimSpace(string(password)), name, "data:image/jpeg;base64," + avatar, "")
  if err != nil {
    fmt.Println(err.Error())
  } else {
    fmt.Println("Changed username successfully")
  }
}

//func SBEvent(s *discordgo.Session, e *discordgo.Event) { ApplyFuncRange(len(sb.hooks.OnEvent), func(i int) { if(ProcessModule("", sb.hooks.OnEvent[i])) { sb.hooks.OnEvent[i].OnEvent(s, e) } }) }
func SBReady(s *discordgo.Session, r *discordgo.Ready) {
  fmt.Println("Ready message receieved, waiting for guilds...")
  go SwapStatusLoop()
  sb.SelfID = r.User.ID
  
  // Only used to change sweetiebot's name or avatar
  //ChangeBotName(s, "Sweetie", "avatar.jpg")
}

func AttachToGuild(g *discordgo.Guild) {
  if sb.initialized {
    sb.log.Log("Multiple initialization detected - updating guild only")
    ProcessGuild(g);
    
    for _, v := range g.Members {
      ProcessMember(v)
    }
    return
  }
  sb.initialized = true
  fmt.Println("Initializing...")
  ProcessGuild(g);
  
  for _, v := range g.Members {
    ProcessMember(v)
  }
  
  episodegencommand := &EpisodeGenCommand{}
  sb.emotemodule = &EmoteModule{}
  spoilermodule := &SpoilerModule{}
  wittymodule := &WittyModule{}
  sb.modules = make([]Module, 0, 6)
  sb.modules = append(sb.modules, &SpamModule{})
  sb.modules = append(sb.modules, &PingModule{})
  sb.modules = append(sb.modules, sb.emotemodule)
  sb.modules = append(sb.modules, wittymodule)
  sb.modules = append(sb.modules, &BoredModule{Episodegen: episodegencommand})
  sb.modules = append(sb.modules, spoilermodule)
  
  for _, v := range sb.modules {
    v.Register(&sb.hooks)
  }
  
  addfuncmap := map[string]func(string)string{
    "emote": func(arg string) string { 
      r := sb.emotemodule.UpdateRegex()
      if !r {
        delete(sb.config.Collections["emote"], arg)
        sb.emotemodule.UpdateRegex()
        return "```Failed to ban " + arg + " because regex compilation failed.```"
      }
      return "```Banned " + arg + " and recompiled the emote regex.```"  
    },
    "spoiler": func(arg string) string { 
      r := spoilermodule.UpdateRegex()
      if !r {
        delete(sb.config.Collections["spoiler"], arg)
        spoilermodule.UpdateRegex()
        return "```Failed to ban " + arg + " because regex compilation failed.```"
      }
      return "```Banned " + arg + " and recompiled the spoiler regex.```"
    },
  }
  removefuncmap := map[string]func(string)string{
    "emote": func(arg string) string { 
      sb.emotemodule.UpdateRegex()
      return "```Unbanned " + arg + " and recompiled the emote regex.```"
    },
    "spoiler": func(arg string) string { 
      spoilermodule.UpdateRegex()
      return "```Unbanned " + arg + " and recompiled the spoiler regex.```"
    },
  }
  // We have to initialize commands and modules up here because they depend on the discord channel state
  sb.AddCommand(&AddCommand{addfuncmap})
  sb.AddCommand(&RemoveCommand{removefuncmap})
  sb.AddCommand(&CollectionsCommand{})
  sb.AddCommand(&EchoCommand{})
  sb.AddCommand(&HelpCommand{})
  sb.AddCommand(&NewUsersCommand{})
  sb.AddCommand(&EnableCommand{})
  sb.AddCommand(&DisableCommand{})
  sb.AddCommand(&UpdateCommand{})
  sb.AddCommand(&AKACommand{})
  sb.AddCommand(&AboutCommand{})
  sb.AddCommand(&LastPingCommand{})
  sb.AddCommand(&SetConfigCommand{})
  sb.AddCommand(&GetConfigCommand{})
  sb.AddCommand(&LastSeenCommand{})
  sb.AddCommand(&DumpTablesCommand{})
  sb.AddCommand(episodegencommand)
  sb.AddCommand(&QuoteCommand{})
  sb.AddCommand(&ShipCommand{})
  sb.AddCommand(&AddWitCommand{wittymodule})
  sb.AddCommand(&RemoveWitCommand{wittymodule})
  sb.AddCommand(&SearchCommand{emotes: sb.emotemodule, statements: make(map[string][]*sql.Stmt)})
  sb.AddCommand(&SetStatusCommand{})
  sb.AddCommand(&AddGroupCommand{})
  sb.AddCommand(&JoinGroupCommand{})
  sb.AddCommand(&ListGroupCommand{})
  sb.AddCommand(&LeaveGroupCommand{})
  sb.AddCommand(&PingCommand{})
  sb.AddCommand(&PurgeGroupCommand{})
  sb.AddCommand(&BestPonyCommand{})
  sb.AddCommand(&BanCommand{})
  sb.AddCommand(&DropCommand{})
  sb.AddCommand(&GiveCommand{})
  sb.AddCommand(&ListCommand{})
  sb.AddCommand(&FightCommand{"",0})
  sb.AddCommand(&CuteCommand{0})

  go IdleCheckLoop()
  
  debug := ". \n\n"
  if sb.config.Debug {
    debug = ".\n[DEBUG BUILD]\n\n"
  }
  sb.log.Log("[](/sbload)\n Sweetiebot version ", sb.version, " successfully loaded on ", g.Name, debug, GetActiveModules(), "\n\n", GetActiveCommands());
}

func SBTypingStart(s *discordgo.Session, t *discordgo.TypingStart) { ApplyFuncRange(len(sb.hooks.OnTypingStart), func(i int) { if ProcessModule("", sb.hooks.OnTypingStart[i]) { sb.hooks.OnTypingStart[i].OnTypingStart(s, t) } }) }
func SBMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
  if m.Author == nil { // This shouldn't ever happen but we check for it anyway
    return
  }
  
  ch, err := sb.dg.State.Channel(m.ChannelID)
  sb.log.LogError("Error retrieving channel ID " + m.ChannelID + ": ", err)
  private := true
  if err == nil { private = ch.IsPrivate } // Because of the magic of web development, we can get a message BEFORE the "channel created" packet for the channel being used by that message.
  
  cid := SBatoi(m.ChannelID)
  if cid != sb.config.LogChannel && !private { // Log this message provided it wasn't sent to the bot-log channel or in a PM
    sb.db.AddMessage(SBatoi(m.ID), SBatoi(m.Author.ID), m.ContentWithMentionsReplaced(), cid, m.MentionEveryone) 
  }
  if m.Author.ID == sb.SelfID || cid == sb.config.LogChannel { // ALWAYS discard any of our own messages or our log messages before analysis.
    SBAddPings(m.Message) // If we're discarding a message we still need to add any pings to the ping table
    return
  }
  
  if boolXOR(sb.config.Debug, m.ChannelID == sb.DebugChannelID) { // debug builds only respond to the debug channel, and release builds ignore it
    return
  }
  
  // Check if this is a command. If it is, process it as a command, otherwise process it with our modules.
  if len(m.Content) > 1 && m.Content[0] == '!' && (len(m.Content) < 2 || m.Content[1] != '!') { // We check for > 1 here because a single character can't possibly be a valid command
    t := time.Now().UTC().Unix()
    
    _, isfree := sb.config.FreeChannels[m.ChannelID]
    if err != nil || (!private && m.ChannelID != sb.DebugChannelID && !isfree) { // Private channels are not limited, nor is the debug channel
      if sb.commandlimit.check(sb.config.Commandperduration, sb.config.Commandmaxduration, t) { // if we've hit the saturation limit, post an error (which itself will only post if the error saturation limit hasn't been hit)
        sb.log.Error(m.ChannelID, "You can't input more than 3 commands every 30 seconds!")
        return
      }
      sb.commandlimit.append(t)
    }
    
    isSBowner := SBatoi(m.Author.ID) == sb.OwnerID
    ignore := false
    ApplyFuncRange(len(sb.hooks.OnCommand), func(i int) { if ProcessModule(m.ChannelID, sb.hooks.OnCommand[i]) { ignore = ignore || sb.hooks.OnCommand[i].OnCommand(s, m.Message) } })
    if ignore && !isSBowner { // if true, a module wants us to ignore this command
      return
    }
    
    args := ParseArguments(m.Content[1:])
    arg := strings.ToLower(args[0])
    alias, ok := sb.config.Aliases[arg]
    if ok { arg = alias }
    c, ok := sb.commands[arg]    
    if ok {
      cch := sb.config.Command_channels[c.Name()]
      _, disabled := sb.config.Command_disabled[c.Name()]
      if disabled { return }
      if !private && len(cch) > 0 {
        _, ok = cch[m.ChannelID]
        if !ok {
          return
        }
      }
      if !isSBowner && !UserHasAnyRole(m.Author.ID, sb.config.Command_roles[c.Name()]) {
        sb.log.Error(m.ChannelID, "You don't have permission to run this command! Allowed Roles: " + GetRoles(c))
        return
      }
      result, usepm := c.Process(args[1:], m.Message)
      if len(result) > 0 {
        targetchannel := m.ChannelID
        if usepm && !private {
          channel, err := s.UserChannelCreate(m.Author.ID)
          sb.log.LogError("Error opening private channel: ", err);
          if err == nil {
            targetchannel = channel.ID
            if rand.Float32() < 0.01 {
              sb.SendMessage(m.ChannelID, "Check your ~~privilege~~ Private Messages for my reply!")
            } else {
              sb.SendMessage(m.ChannelID, "```Check your Private Messages for my reply!```")
            }
          }
        } 
        
        for len(result) > 1999 { // discord has a 2000 character limit
          if result[0:3] == "```" {
            index := strings.LastIndex(result[:1996], "\n")
            if index < 0 { index = 1996 }
            sb.SendMessage(targetchannel, result[:index] + "```")
            result = "```" + result[index:]
          } else {
            index := strings.LastIndex(result[:1999], "\n")
            if index < 0 { index = 1999 }
            sb.SendMessage(targetchannel, result[:index])
            result = result[index:]
          }
        }
        sb.SendMessage(targetchannel, result)
      }
    } else {
      if args[0] != "airhorn" {
        sb.log.Error(m.ChannelID, "Sorry, " + args[0] + " is not a valid command.\nFor a list of valid commands, type !help.")
      }
    }
  } else {
    ApplyFuncRange(len(sb.hooks.OnMessageCreate), func(i int) { if ProcessModule(m.ChannelID, sb.hooks.OnMessageCreate[i]) { sb.hooks.OnMessageCreate[i].OnMessageCreate(s, m.Message) } })
  }  
}

func SBMessageUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) {
  if boolXOR(sb.config.Debug, m.ChannelID == sb.DebugChannelID) { return }
  if m.Author == nil { // Discord sends an update message with an empty author when certain media links are posted
    return
  }
  cid := SBatoi(m.ChannelID)
  if cid != sb.config.LogChannel { // Always ignore messages from the log channel
    sb.db.AddMessage(SBatoi(m.ID), SBatoi(m.Author.ID), m.ContentWithMentionsReplaced(), cid, m.MentionEveryone) 
  }
  ApplyFuncRange(len(sb.hooks.OnMessageUpdate), func(i int) { if ProcessModule(m.ChannelID, sb.hooks.OnMessageUpdate[i]) { sb.hooks.OnMessageUpdate[i].OnMessageUpdate(s, m.Message) } })
}
func SBMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
  if boolXOR(sb.config.Debug, m.ChannelID == sb.DebugChannelID) { return }
  ApplyFuncRange(len(sb.hooks.OnMessageDelete), func(i int) { if ProcessModule(m.ChannelID, sb.hooks.OnMessageDelete[i]) { sb.hooks.OnMessageDelete[i].OnMessageDelete(s, m.Message) } })
}
func SBMessageAck(s *discordgo.Session, m *discordgo.MessageAck) { ApplyFuncRange(len(sb.hooks.OnMessageAck), func(i int) { if ProcessModule(m.ChannelID, sb.hooks.OnMessageAck[i]) { sb.hooks.OnMessageAck[i].OnMessageAck(s, m) } }) }
func SBUserUpdate(s *discordgo.Session, m *discordgo.UserUpdate) { ProcessUser(m.User); ApplyFuncRange(len(sb.hooks.OnUserUpdate), func(i int) { if ProcessModule("", sb.hooks.OnUserUpdate[i]) { sb.hooks.OnUserUpdate[i].OnUserUpdate(s, m.User) } }) }
func SBUserSettingsUpdate(s *discordgo.Session, m *discordgo.UserSettingsUpdate) { fmt.Println("OnUserSettingsUpdate called") }
func SBPresenceUpdate(s *discordgo.Session, m *discordgo.PresenceUpdate) { ProcessUser(m.User); ApplyFuncRange(len(sb.hooks.OnPresenceUpdate), func(i int) { if ProcessModule("", sb.hooks.OnPresenceUpdate[i]) { sb.hooks.OnPresenceUpdate[i].OnPresenceUpdate(s, m) } }) }
func SBVoiceStateUpdate(s *discordgo.Session, m *discordgo.VoiceStateUpdate) { ApplyFuncRange(len(sb.hooks.OnVoiceStateUpdate), func(i int) { if ProcessModule("", sb.hooks.OnVoiceStateUpdate[i]) { sb.hooks.OnVoiceStateUpdate[i].OnVoiceStateUpdate(s, m.VoiceState) } }) }
func SBGuildUpdate(s *discordgo.Session, m *discordgo.GuildUpdate) {
  sb.log.Log("Guild update detected, updating ", m.Name)
  ProcessGuild(m.Guild)
  ApplyFuncRange(len(sb.hooks.OnGuildUpdate), func(i int) { if ProcessModule("", sb.hooks.OnGuildUpdate[i]) { sb.hooks.OnGuildUpdate[i].OnGuildUpdate(s, m.Guild) } })
}
func SBGuildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) { ProcessMember(m.Member); ApplyFuncRange(len(sb.hooks.OnGuildMemberAdd), func(i int) { if ProcessModule("", sb.hooks.OnGuildMemberAdd[i]) { sb.hooks.OnGuildMemberAdd[i].OnGuildMemberAdd(s, m.Member) } }) }
func SBGuildMemberRemove(s *discordgo.Session, m *discordgo.GuildMemberRemove) { ApplyFuncRange(len(sb.hooks.OnGuildMemberRemove), func(i int) { if ProcessModule("", sb.hooks.OnGuildMemberRemove[i]) { sb.hooks.OnGuildMemberRemove[i].OnGuildMemberRemove(s, m.Member) } }) }
func SBGuildMemberUpdate(s *discordgo.Session, m *discordgo.GuildMemberUpdate) { ProcessMember(m.Member); ApplyFuncRange(len(sb.hooks.OnGuildMemberUpdate), func(i int) { if ProcessModule("", sb.hooks.OnGuildMemberUpdate[i]) { sb.hooks.OnGuildMemberUpdate[i].OnGuildMemberUpdate(s, m.Member) } }) }
func SBGuildBanAdd(s *discordgo.Session, m *discordgo.GuildBanAdd) { ApplyFuncRange(len(sb.hooks.OnGuildBanAdd), func(i int) { if ProcessModule("", sb.hooks.OnGuildBanAdd[i]) { sb.hooks.OnGuildBanAdd[i].OnGuildBanAdd(s, m.GuildBan) } }) }
func SBGuildBanRemove(s *discordgo.Session, m *discordgo.GuildBanRemove) { ApplyFuncRange(len(sb.hooks.OnGuildBanRemove), func(i int) { if ProcessModule("", sb.hooks.OnGuildBanRemove[i]) { sb.hooks.OnGuildBanRemove[i].OnGuildBanRemove(s, m.GuildBan) } }) }
func SBGuildCreate(s *discordgo.Session, m *discordgo.GuildCreate) { ProcessGuildCreate(m.Guild) }

func ProcessUser(u *discordgo.User) uint64 {
  id := SBatoi(u.ID)
  sb.db.AddUser(id, u.Email, u.Username, u.Avatar, u.Verified)
  return id
}

func ProcessMember(u *discordgo.Member) {
  ProcessUser(u.User)
  
  if len(u.JoinedAt) > 0 { // Parse join date and update user table only if it is less than our current first seen date.
    t, err := time.Parse(time.RFC3339Nano, u.JoinedAt)
    if err == nil {
      sb.db.UpdateUserJoinTime(SBatoi(u.User.ID), t)
    } else {
      fmt.Println(err.Error())
    }
  }
}

func ProcessGuildCreate(g *discordgo.Guild) {
  AttachToGuild(g);
}

func ProcessGuild(g *discordgo.Guild) {
  sb.Guild = g
  
  for _, v := range g.Channels {
    switch v.Name {
      case "bot-debug":
        sb.DebugChannelID = v.ID
    }
  }
}

func FindChannelID(name string) string {
  channels := sb.dg.State.Guilds[0].Channels 
  for _, v := range channels {
    if v.Name == name {
      return v.ID
    }
  }
  
  return ""
}

func ApplyFuncRange(length int, fn func(i int)) {
  for i := 0; i < length; i++ { fn(i) }
}

func IdleCheckLoop() {
  for !sb.quit {
    ids := sb.Guild.Channels
    if sb.config.Debug { // override this in debug mode
      c, err := sb.dg.State.Channel(sb.DebugChannelID)
      if err == nil { ids = []*discordgo.Channel{c} }
    } 
    for _, id := range ids {
      t := sb.db.GetLatestMessage(SBatoi(id.ID))
      diff := SinceUTC(t);
      ApplyFuncRange(len(sb.hooks.OnIdle), func(i int) {
        if ProcessModule("", sb.hooks.OnIdle[i]) && diff >= (time.Duration(sb.hooks.OnIdle[i].IdlePeriod())*time.Second) {
          sb.hooks.OnIdle[i].OnIdle(sb.dg, id)
          }
        })
    }
    time.Sleep(30*time.Second)
  }  
}

func WaitForInput() {
	var input string
	fmt.Scanln(&input)
	sb.quit = true
}

func Initialize(Token string) {  
  dbauth, _ := ioutil.ReadFile("db.auth")
  config, _ := ioutil.ReadFile("config.json")

  sb = &SweetieBot{
    version: "0.7.0",
    commands: make(map[string]Command),
    log: &Log{0},
    commandlimit: &SaturationLimit{[]int64{}, 0, AtomicFlag{0}},
    initialized: false,
    OwnerID: 95585199324143616,
    //OwnerID: 0,
  }
  
  rand.Intn(10)
  for i := 0; i < 20 + rand.Intn(20); i++ { rand.Intn(50) }

  errjson := json.Unmarshal(config, &sb.config)
  if errjson != nil { fmt.Println("Error reading config file: ", errjson.Error()) }
  //fmt.Println("Config settings: ", sb.config)
  
  sb.commandlimit.times = make([]int64, sb.config.Commandperduration*2, sb.config.Commandperduration*2);
  
  db, err := DB_Load(sb.log, "mysql", strings.TrimSpace(string(dbauth)))
  if err != nil { 
    fmt.Println("Error loading database", err.Error())
    return 
  }
  
  sb.db = db 
  sb.dg, err = discordgo.New(Token)
  if err != nil {
    fmt.Println("Error creating discord session", err.Error())
    return
  }
  
  if len(sb.config.Collections) == 0 {
    sb.config.Collections = make(map[string]map[string]bool);
  }
  collections := []string{"emote", "bored", "cute", "status", "spoiler", "bucket"};
  for _, v := range collections {
    _, ok := sb.config.Collections[v]
    if !ok {
      sb.config.Collections[v] = make(map[string]bool)
    }
  }

  sb.dg.AddHandler(SBReady)
  sb.dg.AddHandler(SBTypingStart)
  sb.dg.AddHandler(SBMessageCreate)
  sb.dg.AddHandler(SBMessageUpdate)
  sb.dg.AddHandler(SBMessageDelete)
  sb.dg.AddHandler(SBMessageAck)
  sb.dg.AddHandler(SBUserUpdate)
  sb.dg.AddHandler(SBUserSettingsUpdate)
  sb.dg.AddHandler(SBPresenceUpdate)
  sb.dg.AddHandler(SBVoiceStateUpdate)
  sb.dg.AddHandler(SBGuildUpdate)
  sb.dg.AddHandler(SBGuildMemberAdd)
  sb.dg.AddHandler(SBGuildMemberRemove)
  sb.dg.AddHandler(SBGuildMemberUpdate)
  sb.dg.AddHandler(SBGuildBanAdd)
  sb.dg.AddHandler(SBGuildBanRemove)
  sb.dg.AddHandler(SBGuildCreate)
  
  sb.db.LoadStatements()
  sb.log.Log("Finished loading database statements")
  
  //BuildMarkov(1, 1)
  //return
  err = sb.dg.Open()
  if err == nil {
    fmt.Println("Connection established");
    
    if sb.config.Debug { // The server does not necessarily tie a standard input to the program
      go WaitForInput()
    }  
    for !sb.quit { time.Sleep(400 * time.Millisecond) }
  } else {
    sb.log.LogError("Error opening websocket connection: ", err);
  }
  
  fmt.Println("Sweetiebot quitting");
  sb.db.Close();
}