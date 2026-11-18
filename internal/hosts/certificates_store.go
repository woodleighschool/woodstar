package hosts

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ReplaceCertificates(ctx context.Context, hostID int64, certificates []HostCertificate) error {
	rows := make([]hostCertificateRow, len(certificates))
	for i, cert := range certificates {
		rows[i] = certificateRow(cert)
	}
	return s.reconcileHostInventory(ctx, hostID, reconcileCertificatesSQL, rows)
}

const reconcileCertificatesSQL = `
WITH observed AS (
    SELECT DISTINCT ON (sha1, source, username) *
    FROM jsonb_populate_recordset(NULL::host_certificates, $2::jsonb) WITH ORDINALITY
    WHERE sha1 <> ''
    ORDER BY sha1, source, username, ordinality DESC
), removed AS (
    DELETE FROM host_certificates t
    WHERE t.host_id = $1
      AND NOT EXISTS (SELECT FROM observed s WHERE t.sha1 = s.sha1 AND t.source = s.source AND t.username = s.username)
)
MERGE INTO host_certificates t
USING observed s ON t.host_id = $1 AND t.sha1 = s.sha1 AND t.source = s.source AND t.username = s.username
WHEN MATCHED AND (
    t.common_name, t.subject_country, t.subject_organization, t.subject_organizational_unit,
    t.subject_common_name, t.issuer_country, t.issuer_organization, t.issuer_organizational_unit,
    t.issuer_common_name, t.key_algorithm, t.key_strength, t.key_usage, t.signing_algorithm,
    t.not_valid_after, t.not_valid_before, t.serial, t.certificate_authority, t.path
) IS DISTINCT FROM (
    s.common_name, s.subject_country, s.subject_organization, s.subject_organizational_unit,
    s.subject_common_name, s.issuer_country, s.issuer_organization, s.issuer_organizational_unit,
    s.issuer_common_name, s.key_algorithm, s.key_strength, s.key_usage, s.signing_algorithm,
    s.not_valid_after, s.not_valid_before, s.serial, s.certificate_authority, s.path
) THEN UPDATE SET
    (common_name, subject_country, subject_organization, subject_organizational_unit,
     subject_common_name, issuer_country, issuer_organization, issuer_organizational_unit,
     issuer_common_name, key_algorithm, key_strength, key_usage, signing_algorithm,
     not_valid_after, not_valid_before, serial, certificate_authority, path) =
    (s.common_name, s.subject_country, s.subject_organization, s.subject_organizational_unit,
     s.subject_common_name, s.issuer_country, s.issuer_organization, s.issuer_organizational_unit,
     s.issuer_common_name, s.key_algorithm, s.key_strength, s.key_usage, s.signing_algorithm,
     s.not_valid_after, s.not_valid_before, s.serial, s.certificate_authority, s.path)
WHEN NOT MATCHED THEN INSERT (
    host_id, sha1, source, username, common_name, subject_country, subject_organization,
    subject_organizational_unit, subject_common_name, issuer_country, issuer_organization,
    issuer_organizational_unit, issuer_common_name, key_algorithm, key_strength, key_usage,
    signing_algorithm, not_valid_after, not_valid_before, serial, certificate_authority, path
) VALUES (
    $1, s.sha1, s.source, s.username, s.common_name, s.subject_country, s.subject_organization,
    s.subject_organizational_unit, s.subject_common_name, s.issuer_country, s.issuer_organization,
    s.issuer_organizational_unit, s.issuer_common_name, s.key_algorithm, s.key_strength,
    s.key_usage, s.signing_algorithm, s.not_valid_after, s.not_valid_before, s.serial,
    s.certificate_authority, s.path
)`

func (s *Store) ListCertificates(ctx context.Context, hostID int64) ([]HostCertificate, error) {
	rows, err := s.pool.Query(ctx, `
SELECT
	id, host_id, sha1, common_name,
	subject_country, subject_organization, subject_organizational_unit, subject_common_name,
	issuer_country, issuer_organization, issuer_organizational_unit, issuer_common_name,
	key_algorithm, key_strength, key_usage, signing_algorithm,
	not_valid_after, not_valid_before, serial, certificate_authority,
	source, username, path
FROM host_certificates
WHERE host_id = $1
ORDER BY common_name, sha1, id`,
		hostID,
	)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[hostCertificateRow])
	if err != nil {
		return nil, err
	}
	certificates := make([]HostCertificate, len(records))
	for i, record := range records {
		certificates[i] = hostCertificateFromRow(record)
	}
	return certificates, nil
}

func certificateRow(certificate HostCertificate) hostCertificateRow {
	return hostCertificateRow{
		SHA1:                      certificate.SHA1,
		CommonName:                certificate.CommonName,
		SubjectCountry:            certificate.Subject.Country,
		SubjectOrganization:       certificate.Subject.Organization,
		SubjectOrganizationalUnit: certificate.Subject.OrganizationalUnit,
		SubjectCommonName:         certificate.Subject.CommonName,
		IssuerCountry:             certificate.Issuer.Country,
		IssuerOrganization:        certificate.Issuer.Organization,
		IssuerOrganizationalUnit:  certificate.Issuer.OrganizationalUnit,
		IssuerCommonName:          certificate.Issuer.CommonName,
		KeyAlgorithm:              certificate.KeyAlgorithm,
		KeyStrength:               certificate.KeyStrength,
		KeyUsage:                  certificate.KeyUsage,
		SigningAlgorithm:          certificate.SigningAlgorithm,
		NotValidAfter:             certificate.NotValidAfter,
		NotValidBefore:            certificate.NotValidBefore,
		Serial:                    certificate.Serial,
		CertificateAuthority:      certificate.CertificateAuthority,
		Source:                    certificate.Source,
		Username:                  certificate.Username,
		Path:                      certificate.Path,
	}
}

type hostCertificateRow struct {
	ID                        int64      `json:"-"`
	HostID                    int64      `json:"-"`
	SHA1                      string     `json:"sha1"`
	CommonName                string     `json:"common_name"`
	SubjectCountry            string     `json:"subject_country"`
	SubjectOrganization       string     `json:"subject_organization"`
	SubjectOrganizationalUnit string     `json:"subject_organizational_unit"`
	SubjectCommonName         string     `json:"subject_common_name"`
	IssuerCountry             string     `json:"issuer_country"`
	IssuerOrganization        string     `json:"issuer_organization"`
	IssuerOrganizationalUnit  string     `json:"issuer_organizational_unit"`
	IssuerCommonName          string     `json:"issuer_common_name"`
	KeyAlgorithm              string     `json:"key_algorithm"`
	KeyStrength               *int32     `json:"key_strength"`
	KeyUsage                  string     `json:"key_usage"`
	SigningAlgorithm          string     `json:"signing_algorithm"`
	NotValidAfter             *time.Time `json:"not_valid_after"`
	NotValidBefore            *time.Time `json:"not_valid_before"`
	Serial                    string     `json:"serial"`
	CertificateAuthority      bool       `json:"certificate_authority"`
	Source                    string     `json:"source"`
	Username                  string     `json:"username"`
	Path                      string     `json:"path"`
}

func hostCertificateFromRow(row hostCertificateRow) HostCertificate {
	return HostCertificate{
		ID:         row.ID,
		HostID:     row.HostID,
		SHA1:       row.SHA1,
		CommonName: row.CommonName,
		Subject: CertificateName{
			Country:            row.SubjectCountry,
			Organization:       row.SubjectOrganization,
			OrganizationalUnit: row.SubjectOrganizationalUnit,
			CommonName:         row.SubjectCommonName,
		},
		Issuer: CertificateName{
			Country:            row.IssuerCountry,
			Organization:       row.IssuerOrganization,
			OrganizationalUnit: row.IssuerOrganizationalUnit,
			CommonName:         row.IssuerCommonName,
		},
		KeyAlgorithm:         row.KeyAlgorithm,
		KeyStrength:          row.KeyStrength,
		KeyUsage:             row.KeyUsage,
		SigningAlgorithm:     row.SigningAlgorithm,
		NotValidAfter:        row.NotValidAfter,
		NotValidBefore:       row.NotValidBefore,
		Serial:               row.Serial,
		CertificateAuthority: row.CertificateAuthority,
		Source:               row.Source,
		Username:             row.Username,
		Path:                 row.Path,
	}
}
