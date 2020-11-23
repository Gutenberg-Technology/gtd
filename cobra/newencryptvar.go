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
	viper.AutomaticEnv() // read in environment variables that match
	// if a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file", viper.ConfigFileUsed())
	}

	if viper.GetString("shared_secret") == "" {
		//generate a shared secret

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
		// sharedSecret := viper.GetString("shared_secret")
		// fmt.Println(Decrypt([]byte(viper.GetString("mysecret")), []byte(sharedSecret)))
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

	// text := []byte("My super Secret Password")
	// key := []byte("passphrarsewichneedstobe32bytes!")

	c, err := aes.NewCipher(key)

	if err != nil {
		fmt.Println(err)
	}

	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	// if any error generating new GCM
	// handle them
	if err != nil {
		fmt.Println(err)
	}
	// creates a new byte array the size of the nonce
	// which must be passed to Seal

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		fmt.Println(err)
	}
	// here we encrypt our text using the Seal function
	// Seal encrypts and authenticates plaintext, authenticates the
	// additional data and appends the result to dst, returning the updated
	// slice. The nonce must be NonceSize() bytes long and unique for all
	// time, for a given key.
	encrypted := gcm.Seal(nonce, nonce, text, nil)
	// err = ioutil.WriteFile("myfyle.data", gcm.Seal(nonce, nonce, text, nil), 0777)

	if err != nil {
		fmt.Println(err)
	}

	return string(encrypted)

}

func assertAvailablePRNG() {
	// Assert that a cryptographically secure PRNG is available.
	// Panic otherwise.
	buf := make([]byte, 1)

	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		panic(fmt.Sprintf("crypto/rand is unavailable: Read() failed with %#v", err))
	}
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
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
