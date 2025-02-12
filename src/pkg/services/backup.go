package service

import (
	"context"
	"fmt"
	"helm-portal/config"
	"helm-portal/pkg/storage"
	"io"
	"os"
	"path/filepath"

	gcs "cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

// BackupService handles backup operations to cloud providers
type BackupService struct {
	pathManager *storage.PathManager
	config      *config.Config
	log         *logrus.Logger
	awsSession  *session.Session
	s3Client    *s3.S3
	gcsClient   *gcs.Client
}

// NewBackupService creates a new backup service
func NewBackupService(config *config.Config, log *logrus.Logger) *BackupService {

	log.SetLevel(logrus.DebugLevel)

	log.WithFields(logrus.Fields{
		"gcp_bucket":  config.Backup.GCP.Bucket,
		"gcp_project": config.Backup.GCP.ProjectID,
	}).Debug("🔍 Configuration loaded")

	srv := &BackupService{
		pathManager: storage.NewPathManager(config.Storage.Path, log),
		config:      config,
		log:         log,
	}

	// Initialize cloud provider client based on config
	if config.Backup.AWS.Bucket != "" {
		sess, err := session.NewSession(&aws.Config{
			Region:      aws.String(config.Backup.AWS.Region),
			Credentials: credentials.NewStaticCredentials(config.Backup.AWS.AccessKeyID, config.Backup.AWS.SecretAccessKey, ""),
		})
		if err != nil {
			log.WithError(err).Error("❌ Failed to initialize AWS session")
		}
		srv.awsSession = sess
		srv.s3Client = s3.New(sess)
	} else if config.Backup.GCP.Bucket != "" {
		log.WithFields(logrus.Fields{
			"gcp_bucket":      config.Backup.GCP.Bucket,
			"gcp_project":     config.Backup.GCP.ProjectID,
			"gcp_credentials": config.Backup.GCP.CredentialsFile,
		}).Info("🔍 Checking GCP configuration")

		// Vérification que le fichier de credentials existe
		if _, err := os.Stat(config.Backup.GCP.CredentialsFile); err != nil {
			log.WithError(err).Error("❌ GCP credentials file not found")
			return srv
		}

		if config.Backup.GCP.Bucket != "" {
			log.Info("🔧 Initializing GCP client")
			ctx := context.Background()
			client, err := gcs.NewClient(ctx, option.WithCredentialsFile(config.Backup.GCP.CredentialsFile))
			if err != nil {
				log.WithError(err).Error("❌ Failed to initialize GCP client")
				return srv
			}
			srv.gcsClient = client

			// Vérifier que le bucket existe
			_, err = client.Bucket(config.Backup.GCP.Bucket).Attrs(ctx)
			if err != nil {
				log.WithError(err).Error("❌ Failed to access GCP bucket")
				return srv
			}

			log.Info("✅ GCP client initialized successfully")
		}

	}

	return srv
}

// Backup performs backup to configured cloud provider
func (s *BackupService) Backup() error {
	s.log.Info("🚀 Starting backup process")

	sourcePath := s.pathManager.GetBasePath()

	if s.awsSession != nil {
		return s.backupToAWS(sourcePath)
	} else if s.gcsClient != nil {
		return s.backupToGCP(sourcePath)
	}
	s.log.Error("❌ No cloud provider configured")
	return fmt.Errorf("❌ no cloud provider configured")
}

func (s *BackupService) backupToAWS(sourcePath string) error {
	s.log.Info("📤 Backing up to AWS S3")
	uploader := s3manager.NewUploader(s.awsSession)

	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("❌ failed to open file %s: %w", path, err)
		}
		defer file.Close()

		relPath, _ := filepath.Rel(sourcePath, path)
		_, err = uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(s.config.Backup.AWS.Bucket),
			Key:    aws.String(relPath),
			Body:   file,
		})

		if err != nil {
			s.log.WithError(err).Errorf("❌ Failed to upload %s", relPath)
			return err
		}

		s.log.Infof("✅ Uploaded %s", relPath)
		return nil
	})
}

func (s *BackupService) backupToGCP(sourcePath string) error {
	s.log.Info("📤 Backing up to GCP Storage")
	ctx := context.Background()

	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("❌ failed to open file %s: %w", path, err)
		}
		defer file.Close()

		relPath, _ := filepath.Rel(sourcePath, path)
		obj := s.gcsClient.Bucket(s.config.Backup.GCP.Bucket).Object(relPath)
		writer := obj.NewWriter(ctx)

		if _, err := io.Copy(writer, file); err != nil {
			s.log.WithError(err).Errorf("❌ Failed to upload %s", relPath)
			return err
		}

		if err := writer.Close(); err != nil {
			return err
		}

		s.log.Infof("✅ Uploaded %s", relPath)
		return nil
	})
}

// Restore downloads backup from cloud provider
func (s *BackupService) Restore() error {
	s.log.Info("🔄 Starting restore process")

	if s.awsSession != nil {
		return s.restoreFromAWS()
	} else if s.gcsClient != nil {
		return s.restoreFromGCP()
	}

	return fmt.Errorf("❌ no cloud provider configured")
}

func (s *BackupService) restoreFromGCP() error {
	panic("unimplemented")
}

func (s *BackupService) restoreFromAWS() error {
	panic("unimplemented")
}
