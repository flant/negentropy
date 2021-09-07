package consts

import "time"

// CacheTTL defines how to store values at permanent cache
var CacheTTL = time.Hour * 24 * 14

// SSHWorkdir defines path for storing temporal files of ssh-session
const SSHWorkdir = "/tmp/flint"
