package database

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func (s *SSHConfig) Endpoint() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

func (s *SSHConfig) ClientConfig() (*ssh.ClientConfig, error) {
	var (
		auth ssh.AuthMethod
		err  error
	)

	if strings.HasPrefix(s.PrivateKey, "agent://") {
		auth, err = s.sshAgentAuthMethod(strings.TrimPrefix(s.PrivateKey, "agent://"))
		if err != nil {
			return nil, err
		}
	} else {
		buffer, err := os.ReadFile(s.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("cannot read SSH private key file, PrivateKey=%s, %w", s.PrivateKey, err)
		}

		var key ssh.Signer
		if s.PassPhrase != "" {
			key, err = ssh.ParsePrivateKeyWithPassphrase(buffer, []byte(s.PassPhrase))
			if err != nil {
				return nil, fmt.Errorf("cannot parse SSH private key file with passphrase, PrivateKey=%s, %w", s.PrivateKey, err)
			}
		} else {
			key, err = ssh.ParsePrivateKey(buffer)
			if err != nil {
				return nil, fmt.Errorf("cannot parse SSH private key file, PrivateKey=%s, %w", s.PrivateKey, err)
			}
		}
		auth = ssh.PublicKeys(key)
	}

	sshConfig := &ssh.ClientConfig{
		User:            s.User,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return sshConfig, nil
}

func (s *SSHConfig) sshAgentAuthMethod(selector string) (ssh.AuthMethod, error) {
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock == "" {
		return nil, errors.New("SSH_AUTH_SOCK is not set (ssh-agent is not available)")
	}

	selector = strings.TrimSpace(selector)
	selector = strings.TrimPrefix(selector, "/")

	return ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
		sock := os.Getenv("SSH_AUTH_SOCK")
		if sock == "" {
			return nil, errors.New("SSH_AUTH_SOCK is not set (ssh-agent is not available)")
		}
		conn, err := net.Dial("unix", sock)
		if err != nil {
			return nil, fmt.Errorf("cannot connect to SSH agent, SSH_AUTH_SOCK=%s, %w", sock, err)
		}
		ag := agent.NewClient(conn)
		keys, err := ag.List()
		_ = conn.Close()
		if err != nil {
			return nil, fmt.Errorf("cannot list SSH agent keys, %w", err)
		}
		if len(keys) == 0 {
			return nil, errors.New("no keys available in SSH agent")
		}

		matchesHash := func(sel string, pk ssh.PublicKey) bool {
			if sel == "" {
				return true
			}
			sha := ssh.FingerprintSHA256(pk)
			shaNoPrefix := strings.TrimPrefix(sha, "SHA256:")
			md5 := ssh.FingerprintLegacyMD5(pk)
			return sel == sha || sel == shaNoPrefix || sel == md5
		}

		matchesName := func(sel string, comment string) bool {
			if sel == "" {
				return true
			}
			if comment == sel {
				return true
			}
			return comment != "" && strings.Contains(comment, sel)
		}

		matched := make([]ssh.Signer, 0, len(keys))
		for _, k := range keys {
			pk, err := ssh.ParsePublicKey(k.Blob)
			if err != nil {
				continue
			}
			comment := k.Comment

			if selector == "" || matchesName(selector, comment) || matchesHash(selector, pk) {
				matched = append(matched, &sshAgentSigner{pub: pk})
			}
		}

		if len(matched) == 0 {
			return nil, fmt.Errorf("no matching SSH agent key for selector %q", selector)
		}
		return matched, nil
	}), nil
}

type sshAgentSigner struct {
	pub ssh.PublicKey
}

var _ ssh.Signer = (*sshAgentSigner)(nil)
var _ ssh.AlgorithmSigner = (*sshAgentSigner)(nil)

func (s *sshAgentSigner) PublicKey() ssh.PublicKey { return s.pub }

func (s *sshAgentSigner) Sign(rand io.Reader, data []byte) (*ssh.Signature, error) {
	c, conn, err := dialSSHAgent()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return c.Sign(s.pub, data)
}

func (s *sshAgentSigner) SignWithAlgorithm(rand io.Reader, data []byte, algorithm string) (*ssh.Signature, error) {
	c, conn, err := dialSSHAgent()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	switch algorithm {
	case ssh.KeyAlgoRSASHA256:
		return c.SignWithFlags(s.pub, data, agent.SignatureFlagRsaSha256)
	case ssh.KeyAlgoRSASHA512:
		return c.SignWithFlags(s.pub, data, agent.SignatureFlagRsaSha512)
	default:
		return c.Sign(s.pub, data)
	}
}

func dialSSHAgent() (agent.ExtendedAgent, net.Conn, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, nil, errors.New("SSH_AUTH_SOCK is not set (ssh-agent is not available)")
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to SSH agent, SSH_AUTH_SOCK=%s, %w", sock, err)
	}
	return agent.NewClient(conn), conn, nil
}
