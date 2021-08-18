package system

import (
	"os"
	"strings"
)

// TODO update this and it can be several keys
const DefaultUserKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDT8bXBWcX25++KjVbVcEAPnURTpQ/6nSi0xV+lqn/sOCTJljRYTMk0UbGZHbvVV2edYYpgCkDKOWF9eF5hpjmPasyYoskVMJz9fae1+cl+pmycnGTZLspg0r59Nr/77ryja2KkllY4CXc7zlmu/R+316VgzpWPyR4007lIZl28BAahbdfKJ/jNppkVZL413rSFZ5gTBlMIVmZq1pUvOBtahMrH2KDvWdg0UJWh5qjyHxifC+Lv3vHWlKqwLrgJFQ859V4sbfTqht41Xwt80nkKSA7yOL4FnxFwxJq5DSJb5yGTYntv3TYlwmlZmL3Jg14S3aQzW80IbVuRwnDfwv0l`

const AuthorizedKeysTemplate = `cert-authority,principals="PRINCIPAL" USER_KEY`

func GenerateAuthorizedKeysFile(principal string, userKey string) string {
	if userKey == "" {
		userKey = DefaultUserKey
	}
	content := AuthorizedKeysTemplate
	content = strings.Replace(content, "PRINCIPAL", principal, 1)
	return strings.Replace(content, "USER_KEY", userKey, 1)
}

func CreateAuthorizedKeysFile(filePath string, content string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write([]byte(content))
	if err != nil {
		return err
	}

	return nil
}
