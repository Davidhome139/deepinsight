package main

import (
	"crypto/rand"
	"crypto/rsa"
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

func main() {
	// 生成私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("生成私钥失败: %v", err)
	}

	// 创建证书模板
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:     []string{"localhost"},
	}

	// 生成证书
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		log.Fatalf("生成证书失败: %v", err)
	}

	// 编码私钥为PEM格式
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// 编码证书为PEM格式
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// 确保证书目录存在
	certDir := "backend/certs"
	if err := os.MkdirAll(certDir, 0755); err != nil {
		log.Fatalf("创建证书目录失败: %v", err)
	}

	// 写入私钥文件
	if err := os.WriteFile(fmt.Sprintf("%s/localhost.key", certDir), privateKeyPEM, 0600); err != nil {
		log.Fatalf("写入私钥文件失败: %v", err)
	}

	// 写入证书文件
	if err := os.WriteFile(fmt.Sprintf("%s/localhost.crt", certDir), certPEM, 0644); err != nil {
		log.Fatalf("写入证书文件失败: %v", err)
	}

	fmt.Println("自签名证书和私钥生成成功:")
	fmt.Printf("证书文件: %s/localhost.crt\n", certDir)
	fmt.Printf("私钥文件: %s/localhost.key\n", certDir)
}
