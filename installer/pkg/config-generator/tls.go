package configgenerator

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"

	"github.com/openshift/installer/installer/pkg/copy"
	"github.com/openshift/installer/pkg/asset/tls"
)

const (
	adminCertPath                = "generated/tls/admin.crt"
	adminKeyPath                 = "generated/tls/admin.key"
	aggregatorCACertPath         = "generated/tls/aggregator-ca.crt"
	aggregatorCAKeyPath          = "generated/tls/aggregator-ca.key"
	apiServerCertPath            = "generated/tls/apiserver.crt"
	apiServerKeyPath             = "generated/tls/apiserver.key"
	apiServerProxyCertPath       = "generated/tls/apiserver-proxy.crt"
	apiServerProxyKeyPath        = "generated/tls/apiserver-proxy.key"
	etcdCACertPath               = "generated/tls/etcd-ca.crt"
	etcdCAKeyPath                = "generated/tls/etcd-ca.key"
	etcdClientCertPath           = "generated/tls/etcd-client.crt"
	etcdClientKeyPath            = "generated/tls/etcd-client.key"
	ingressCACertPath            = "generated/tls/ingress-ca.crt"
	ingressCertPath              = "generated/tls/ingress.crt"
	ingressKeyPath               = "generated/tls/ingress.key"
	kubeCACertPath               = "generated/tls/kube-ca.crt"
	kubeCAKeyPath                = "generated/tls/kube-ca.key"
	kubeletCertPath              = "generated/tls/kubelet.crt"
	kubeletKeyPath               = "generated/tls/kubelet.key"
	clusterAPIServerCertPath     = "generated/tls/cluster-apiserver-ca.crt"
	clusterAPIServerKeyPath      = "generated/tls/cluster-apiserver-ca.key"
	osAPIServerCertPath          = "generated/tls/openshift-apiserver.crt"
	osAPIServerKeyPath           = "generated/tls/openshift-apiserver.key"
	rootCACertPath               = "generated/tls/root-ca.crt"
	rootCAKeyPath                = "generated/tls/root-ca.key"
	serviceServingCACertPath     = "generated/tls/service-serving-ca.crt"
	serviceServingCAKeyPath      = "generated/tls/service-serving-ca.key"
	tncCertPath                  = "generated/tls/tnc.crt"
	tncKeyPath                   = "generated/tls/tnc.key"
	serviceAccountPubkeyPath     = "generated/tls/service-account.pub"
	serviceAccountPrivateKeyPath = "generated/tls/service-account.key"
)

// GenerateTLSConfig fetches and validates the TLS cert files
// If no file paths were provided, the certs will be auto-generated
func (c *ConfigGenerator) GenerateTLSConfig(clusterDir string) error {
	var caKey *rsa.PrivateKey
	var caCert *x509.Certificate
	var err error

	if c.CA.RootCAKeyPath == "" && c.CA.RootCACertPath == "" {
		caCert, caKey, err = generateRootCert(clusterDir)
		if err != nil {
			return fmt.Errorf("failed to generate root CA certificate and key pair: %v", err)
		}
	} else {
		// copy key and certificates
		caCert, caKey, err = getCertFiles(clusterDir, c.CA.RootCACertPath, c.CA.RootCAKeyPath)
		if err != nil {
			return fmt.Errorf("failed to process CA certificate and key pair: %v", err)
		}
	}

	// generate kube CA
	cfg := &tls.CertCfg{
		Subject:   pkix.Name{CommonName: "kube-ca", OrganizationalUnit: []string{"bootkube"}},
		KeyUsages: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		Validity:  tls.ValidityTenYears,
		IsCA:      true,
	}
	kubeCAKey, kubeCACert, err := generateCert(clusterDir, caKey, caCert, kubeCAKeyPath, kubeCACertPath, cfg, false)
	if err != nil {
		return fmt.Errorf("failed to generate kubernetes CA: %v", err)
	}

	// generate etcd CA
	cfg = &tls.CertCfg{
		Subject:   pkix.Name{CommonName: "etcd", OrganizationalUnit: []string{"etcd"}},
		KeyUsages: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		IsCA:      true,
		Validity:  tls.ValidityTenYears,
	}
	etcdCAKey, etcdCACert, err := generateCert(clusterDir, caKey, caCert, etcdCAKeyPath, etcdCACertPath, cfg, false)
	if err != nil {
		return fmt.Errorf("failed to generate etcd CA: %v", err)
	}

	if err := copy.Copy(filepath.Join(clusterDir, etcdCAKeyPath), filepath.Join(clusterDir, "generated/tls/etcd-client-ca.key")); err != nil {
		return fmt.Errorf("failed to import kube CA cert into ingress-ca.crt: %v", err)
	}
	if err := copy.Copy(filepath.Join(clusterDir, etcdCACertPath), filepath.Join(clusterDir, "generated/tls/etcd-client-ca.crt")); err != nil {
		return fmt.Errorf("failed to import kube CA cert into ingress-ca.crt: %v", err)
	}

	// generate etcd client certificate
	cfg = &tls.CertCfg{
		Subject:      pkix.Name{CommonName: "etcd", OrganizationalUnit: []string{"etcd"}},
		KeyUsages:    x509.KeyUsageKeyEncipherment,
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Validity:     tls.ValidityTenYears,
	}
	if _, _, err := generateCert(clusterDir, etcdCAKey, etcdCACert, etcdClientKeyPath, etcdClientCertPath, cfg, false); err != nil {
		return fmt.Errorf("failed to generate etcd client certificate: %v", err)
	}

	// generate aggregator CA
	cfg = &tls.CertCfg{
		Subject:   pkix.Name{CommonName: "aggregator", OrganizationalUnit: []string{"bootkube"}},
		KeyUsages: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		Validity:  tls.ValidityTenYears,
		IsCA:      true,
	}
	aggregatorCAKey, aggregatorCACert, err := generateCert(clusterDir, caKey, caCert, aggregatorCAKeyPath, aggregatorCACertPath, cfg, false)
	if err != nil {
		return fmt.Errorf("failed to generate aggregator CA: %v", err)
	}

	// generate service-serving CA
	cfg = &tls.CertCfg{
		Subject:   pkix.Name{CommonName: "service-serving", OrganizationalUnit: []string{"bootkube"}},
		KeyUsages: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		Validity:  tls.ValidityTenYears,
		IsCA:      true,
	}
	if _, _, err := generateCert(clusterDir, caKey, caCert, serviceServingCAKeyPath, serviceServingCACertPath, cfg, false); err != nil {
		return fmt.Errorf("failed to generate service-serving CA: %v", err)
	}

	// Ingress certs
	if err := copy.Copy(filepath.Join(clusterDir, kubeCACertPath), filepath.Join(clusterDir, ingressCACertPath)); err != nil {
		return fmt.Errorf("failed to import kube CA cert into ingress-ca.crt: %v", err)
	}

	baseAddress := c.getBaseAddress()
	cfg = &tls.CertCfg{
		KeyUsages:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames: []string{
			baseAddress,
			fmt.Sprintf("*.%s", baseAddress),
		},
		Subject:  pkix.Name{CommonName: baseAddress, Organization: []string{"ingress"}},
		Validity: tls.ValidityTenYears,
		IsCA:     false,
	}

	if _, _, err := generateCert(clusterDir, kubeCAKey, kubeCACert, ingressKeyPath, ingressCertPath, cfg, true); err != nil {
		return fmt.Errorf("failed to generate ingress CA: %v", err)
	}

	// Kube admin certs
	cfg = &tls.CertCfg{
		KeyUsages:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		Subject:      pkix.Name{CommonName: "system:admin", Organization: []string{"system:masters"}},
		Validity:     tls.ValidityTenYears,
		IsCA:         false,
	}

	if _, _, err = generateCert(clusterDir, kubeCAKey, kubeCACert, adminKeyPath, adminCertPath, cfg, false); err != nil {
		return fmt.Errorf("failed to generate kube admin certificate: %v", err)
	}

	// Kube API server certs
	apiServerAddress, err := cidrhost(c.Cluster.Networking.ServiceCIDR, 1)
	if err != nil {
		return fmt.Errorf("can't resolve api server host address: %v", err)
	}
	cfg = &tls.CertCfg{
		KeyUsages:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		Subject:      pkix.Name{CommonName: "kube-apiserver", Organization: []string{"kube-master"}},
		DNSNames: []string{
			fmt.Sprintf("%s-api.%s", c.Name, c.BaseDomain),
			"kubernetes", "kubernetes.default",
			"kubernetes.default.svc",
			"kubernetes.default.svc.cluster.local",
		},
		Validity:    tls.ValidityTenYears,
		IPAddresses: []net.IP{net.ParseIP(apiServerAddress)},
		IsCA:        false,
	}

	if _, _, err := generateCert(clusterDir, kubeCAKey, kubeCACert, apiServerKeyPath, apiServerCertPath, cfg, true); err != nil {
		return fmt.Errorf("failed to generate kube api server certificate: %v", err)
	}

	// Kube API openshift certs
	cfg = &tls.CertCfg{
		KeyUsages:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		Subject:      pkix.Name{CommonName: "openshift-apiserver", Organization: []string{"kube-master"}},
		DNSNames: []string{
			fmt.Sprintf("%s-api.%s", c.Name, c.BaseDomain),
			"openshift-apiserver",
			"openshift-apiserver.kube-system",
			"openshift-apiserver.kube-system.svc",
			"openshift-apiserver.kube-system.svc.cluster.local",
			"localhost", "127.0.0.1"},
		Validity:    tls.ValidityTenYears,
		IPAddresses: []net.IP{net.ParseIP(apiServerAddress)},
		IsCA:        false,
	}

	if _, _, err := generateCert(clusterDir, aggregatorCAKey, aggregatorCACert, osAPIServerKeyPath, osAPIServerCertPath, cfg, true); err != nil {
		return fmt.Errorf("failed to generate openshift api server certificate: %v", err)
	}

	// Kube API proxy certs
	cfg = &tls.CertCfg{
		KeyUsages:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Subject:      pkix.Name{CommonName: "kube-apiserver-proxy", Organization: []string{"kube-master"}},
		Validity:     tls.ValidityTenYears,
		IsCA:         false,
	}

	if _, _, err := generateCert(clusterDir, aggregatorCAKey, aggregatorCACert, apiServerProxyKeyPath, apiServerProxyCertPath, cfg, false); err != nil {
		return fmt.Errorf("failed to generate kube api proxy certificate: %v", err)
	}

	// Kubelet certs
	cfg = &tls.CertCfg{
		KeyUsages:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Subject:      pkix.Name{CommonName: "system:serviceaccount:kube-system:default", Organization: []string{"system:serviceaccounts:kube-system"}},
		Validity:     tls.ValidityThirtyMinutes,
		IsCA:         false,
	}

	if _, _, err := generateCert(clusterDir, kubeCAKey, kubeCACert, kubeletKeyPath, kubeletCertPath, cfg, false); err != nil {
		return fmt.Errorf("failed to generate kubelet certificate: %v", err)
	}

	// TNC certs
	tncDomain := fmt.Sprintf("%s-tnc.%s", c.Name, c.BaseDomain)
	cfg = &tls.CertCfg{
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{tncDomain},
		Subject:      pkix.Name{CommonName: tncDomain},
		Validity:     tls.ValidityTenYears,
		IsCA:         false,
	}

	if _, _, err := generateCert(clusterDir, caKey, caCert, tncKeyPath, tncCertPath, cfg, false); err != nil {
		return fmt.Errorf("failed to generate tnc certificate: %v", err)
	}

	// Cluster API cert
	cfg = &tls.CertCfg{
		KeyUsages:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		Subject:      pkix.Name{CommonName: "clusterapi", OrganizationalUnit: []string{"bootkube"}},
		DNSNames: []string{
			"clusterapi",
			fmt.Sprintf("clusterapi.%s", maoTargetNamespace),
			fmt.Sprintf("clusterapi.%s.svc", maoTargetNamespace),
			fmt.Sprintf("clusterapi.%s.svc.cluster.local", maoTargetNamespace),
		},
		Validity: tls.ValidityTenYears,
		IsCA:     false,
	}

	if _, _, err := generateCert(clusterDir, aggregatorCAKey, aggregatorCACert, clusterAPIServerKeyPath, clusterAPIServerCertPath, cfg, true); err != nil {
		return fmt.Errorf("failed to generate cluster-apiserver certificate: %v", err)
	}

	// Service Account private and public key.
	svcAccountPrivKey, err := generatePrivateKey(clusterDir, serviceAccountPrivateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to generate service-account private key: %v", err)
	}

	pubkeyPath := filepath.Join(clusterDir, serviceAccountPubkeyPath)
	pubkeyData, err := tls.PublicKeyToPem(&svcAccountPrivKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to generate service-account public key: %v", err)
	}
	if err := ioutil.WriteFile(pubkeyPath, []byte(pubkeyData), 0600); err != nil {
		return fmt.Errorf("failed to write service-account public key: %v", err)
	}

	return nil
}

// generatePrivateKey generates and writes the private key to disk
func generatePrivateKey(clusterDir string, path string) (*rsa.PrivateKey, error) {
	fileTargetPath := filepath.Join(clusterDir, path)
	key, err := tls.PrivateKey()
	if err != nil {
		return nil, fmt.Errorf("error writing private key: %v", err)
	}
	if err := ioutil.WriteFile(fileTargetPath, []byte(tls.PrivateKeyToPem(key)), 0600); err != nil {
		return nil, err
	}
	return key, nil
}

// generateRootCert creates the rootCAKey and rootCACert
func generateRootCert(clusterDir string) (cert *x509.Certificate, key *rsa.PrivateKey, err error) {
	targetKeyPath := filepath.Join(clusterDir, rootCAKeyPath)
	targetCertPath := filepath.Join(clusterDir, rootCACertPath)

	cfg := &tls.CertCfg{
		Subject: pkix.Name{
			CommonName:         "root-ca",
			OrganizationalUnit: []string{"openshift"},
		},
		KeyUsages: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		Validity:  tls.ValidityTenYears,
		IsCA:      true,
	}

	caKey, caCert, err := tls.GenerateRootCertKey(cfg)
	if err != nil {
		return nil, nil, err
	}

	if err := ioutil.WriteFile(targetKeyPath, []byte(tls.PrivateKeyToPem(caKey)), 0600); err != nil {
		return nil, nil, err
	}

	if err := ioutil.WriteFile(targetCertPath, []byte(tls.CertToPem(caCert)), 0666); err != nil {
		return nil, nil, err
	}

	return caCert, caKey, nil
}

// getCertFiles copies the given cert/key files into the generated folder and returns their contents
func getCertFiles(clusterDir string, certPath string, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	keyDst := filepath.Join(clusterDir, rootCAKeyPath)
	if err := copy.Copy(keyPath, keyDst); err != nil {
		return nil, nil, fmt.Errorf("failed to write file: %v", err)
	}

	certDst := filepath.Join(clusterDir, rootCACertPath)
	if err := copy.Copy(certPath, certDst); err != nil {
		return nil, nil, fmt.Errorf("failed to write file: %v", err)
	}
	// content validation occurs in pkg/config/validate.go
	// if it fails here, something went wrong
	certData, err := ioutil.ReadFile(certPath)
	if err != nil {
		panic(err)
	}
	certPem, _ := pem.Decode([]byte(string(certData)))
	keyData, err := ioutil.ReadFile(keyPath)
	if err != nil {
		panic(err)
	}
	keyPem, _ := pem.Decode([]byte(string(keyData)))
	key, err := x509.ParsePKCS1PrivateKey(keyPem.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to process private key: %v", err)
	}
	certs, err := x509.ParseCertificates(certPem.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to process certificate: %v", err)
	}

	return certs[0], key, nil
}

// generateCert creates a key, csr & a signed cert
// If appendCA is true, then also append the CA cert into the result cert.
// This is useful for apiserver and openshift-apiser cert which will be
// authenticated by the kubeconfig using root-ca.
func generateCert(clusterDir string,
	caKey *rsa.PrivateKey,
	caCert *x509.Certificate,
	keyPath string,
	certPath string,
	cfg *tls.CertCfg,
	appendCA bool) (*rsa.PrivateKey, *x509.Certificate, error) {

	targetKeyPath := filepath.Join(clusterDir, keyPath)
	targetCertPath := filepath.Join(clusterDir, certPath)

	key, cert, err := tls.GenerateCert(caKey, caCert, cfg)
	if err != nil {
		return nil, nil, err
	}

	if err := ioutil.WriteFile(targetKeyPath, []byte(tls.PrivateKeyToPem(key)), 0600); err != nil {
		return nil, nil, err
	}

	content := []byte(tls.CertToPem(cert))
	if appendCA {
		content = append(content, '\n')
		content = append(content, []byte(tls.CertToPem(caCert))...)
	}
	if err := ioutil.WriteFile(targetCertPath, content, 0666); err != nil {
		return nil, nil, err
	}

	return key, cert, nil
}
