package cobra

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var key string
var value string

func init() {
	assertAvailablePRNG()
}

func NewEncryptVarCommand(cmd *Command) {

	cobraCmd := &cobra.Command{
		Use:   "add_secret",
		Short: "Encrypt a var on config",

		Run: func(cobraCmd *cobra.Command, args []string) {
			fmt.Println("Encrypt var on config file!")
			addEncryptVar()
		},
		PreRun: func(cobraCmd *cobra.Command, args []string) {
		},
	}

	cobraCmd.Flags().StringVar(&key, "key", "", "key")
	cobraCmd.Flags().StringVar(&value, "value", "", "Value")
	err := cobraCmd.MarkFlagRequired("key")
	if err != nil {
		fmt.Printf("newencryptvar.missing.key err:%v\n", err)
	}

	err = cobraCmd.MarkFlagRequired("value")
	if err != nil {
		fmt.Printf("newencryptvar.missing.key err:%v\n", err)
	}

	cmd.AddCommand(cobraCmd)
}

func addEncryptVar() {

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".gtd")
	}
	viper.SetEnvPrefix("gtd")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file", viper.ConfigFileUsed())
	}

	if viper.GetString("shared_secret") == "" {
		fmt.Println("Missing ENV 'GTD_SHARED_SECRET'")
		randomString, err := GenerateRandomString(32)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("set new Environment 'GTD_SHARED_SECRET=%s\nLenght: %d'", randomString, len(randomString))
		os.Exit(0)
	} else {
		if sharedSecret := viper.GetString("shared_secret"); len(sharedSecret) != 32 {
			fmt.Println("invalid 'GTD_SHARED_SECRET'")
			os.Exit(1)
		}
		if strings.HasPrefix(key, "secret_") {
			viper.Set(key, Encrypt([]byte(value), []byte(viper.GetString("shared_secret"))))

		} else {

			viper.Set(fmt.Sprintf("secret_%s", key), Encrypt([]byte(value), []byte(viper.GetString("shared_secret"))))
		}

		err := viper.WriteConfig()
		if err != nil {
			fmt.Printf("addEncryptVar.WriteConfig. err:%v\n", err)
		}
	}

}

func Decrypt(ciphertext, key []byte) string {

	c, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println(err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		fmt.Println(err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		fmt.Println(err)
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		fmt.Println(err)
	}
	return string(plaintext)

}

func Encrypt(text, key []byte) string {
	c, err := aes.NewCipher(key)

	if err != nil {
		fmt.Println(err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		fmt.Println(err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		fmt.Println(err)
	}
	encrypted := gcm.Seal(nonce, nonce, text, nil)

	if err != nil {
		fmt.Println(err)
	}

	return string(encrypted)

}

func assertAvailablePRNG() {
	buf := make([]byte, 1)

	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		panic(fmt.Sprintf("crypto/rand is unavailable: Read() failed with %#v", err))
	}
}

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func GenerateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-$*%+-;?<>@({[]})"
	bytes, err := GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes), nil
}
