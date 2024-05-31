package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

const (
	DefaultPem = "cert.pem"
	DefaultKey = "cert.key"
)

func generateTLSConfig() (*tls.Config, error) {
	// 读取证书
	certPEM, err := os.ReadFile("cert.pem")
	file, err := os.ReadFile("cert.key")
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %v", err)
	}

	// 解析证书
	certs, err := tls.X509KeyPair(certPEM, file) // 通常私钥也在同一文件中，这里是简化示例
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	// 创建TLS配置
	config := &tls.Config{
		Certificates: []tls.Certificate{certs},
		NextProtos:   []string{"h3-29"}, // 指定QUIC的HTTP/3版本
	}

	return config, nil
}
func CreateESDATLS() {
	// 生成私钥
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Println("私钥生成失败：", err)
		return
	}

	// 生成证书请求
	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "localhost",
			Organization: []string{"Acme Co"},
		},
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
	if err != nil {
		fmt.Println("证书请求生成失败：", err)
		return
	}

	csrFile, err := os.Create("cert.csr")
	if err != nil {
		fmt.Println("无法创建证书请求文件：", err)
		return
	}
	defer csrFile.Close()

	pem.Encode(csrFile, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
	fmt.Println("证书请求生成成功")

	// 生成自签名证书
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		fmt.Println("序列号生成失败：", err)
		return
	}

	now := time.Now()
	// 设置证书有效期
	certTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "localhost",
			Organization: []string{"Acme Co"},
		},
		NotBefore:             now,
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		fmt.Println("证书生成失败：", err)
		return
	}

	certFile, err := os.Create("cert.pem")
	if err != nil {
		fmt.Println("无法创建证书文件：", err)
		return
	}
	defer certFile.Close()
	keyOut, err := os.Create("cert.key")
	if err != nil {
		log.Fatal("Failed to open key.pem for writing:", err)
	}

	key, err := x509.MarshalECPrivateKey(privateKey)
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Headers: nil, Bytes: key})
	defer keyOut.Close()

	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	fmt.Println("证书生成成功")
}
