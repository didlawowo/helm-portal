package service

import (
	"context"
	"fmt"
	"helm-portal/config"
	"helm-portal/pkg/utils"
	"io"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	gcs "cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

type BackupService struct {
	pathManager *utils.PathManager
	config      *config.Config
	log         *utils.Logger
	awsSession  *session.Session
	s3Client    *s3.S3
	gcsClient   *gcs.Client
}

func NewBackupService(config *config.Config, log *utils.Logger) (*BackupService, error) {
	if config == nil {
		return nil, fmt.Errorf("❌ invalid configuration: config is nil")
	}

	if log == nil {
		return nil, fmt.Errorf("❌ logger is nil")
	}

	srv := &BackupService{
		pathManager: utils.NewPathManager(config.Storage.Path, log),
		config:      config,
		log:         log,
	}

	// Initialisation du client cloud
	if config.Backup.AWS.Bucket != "" {
		if err := srv.initAWSClient(); err != nil {
			return nil, fmt.Errorf("❌ failed to initialize AWS client: %w", err)
		}
	} else if config.Backup.GCP.Bucket != "" {
		if err := srv.initGCPClient(); err != nil {
			return nil, fmt.Errorf("❌ failed to initialize GCP client: %w", err)
		}
	} else {
		return nil, fmt.Errorf("❌ no cloud provider configured - please configure either AWS or GCP backup settings")
	}

	return srv, nil
}

func (s *BackupService) initAWSClient() error {
	s.log.WithFunc().WithFields(logrus.Fields{
		"region": s.config.Backup.AWS.Region,
		"bucket": s.config.Backup.AWS.Bucket,
	}).Debug("Initializing AWS client")

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(s.config.Backup.AWS.Region),
		Credentials: credentials.NewStaticCredentials(s.config.Backup.AWS.AccessKeyID, s.config.Backup.AWS.SecretAccessKey, ""),
	})
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %w", err)
	}

	s.awsSession = sess
	s.s3Client = s3.New(sess)
	return nil
}

func (s *BackupService) initGCPClient() error {
	// Ajout de timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Vérification des prérequis
	if s.config.Backup.GCP.Bucket == "" {
		return fmt.Errorf("GCP bucket name is not configured")
	}
	if s.config.Backup.GCP.ProjectID == "" {
		return fmt.Errorf("GCP project ID is not configured")
	}

	// Vérification du fichier de credentials
	if _, err := os.Stat(s.config.Backup.GCP.CredentialsFile); err != nil {
		s.log.WithFunc().WithError(err).WithField("credentialsPath", s.config.Backup.GCP.CredentialsFile).Error("Credentials file check failed")
		return fmt.Errorf("credentials file not found: %w", err)
	}

	// Création du client
	client, err := gcs.NewClient(ctx, option.WithCredentialsFile(s.config.Backup.GCP.CredentialsFile))
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	// En cas d'erreur après ce point, on s'assure de fermer le client
	defer func() {
		if err != nil {
			client.Close()
		}
	}()

	// Vérification du bucket
	bucket := client.Bucket(s.config.Backup.GCP.Bucket)
	attrs, err := bucket.Attrs(ctx)
	if err != nil {
		if err == storage.ErrBucketNotExist {
			s.log.WithFunc().WithField("bucket", s.config.Backup.GCP.Bucket).Error("Bucket does not exist")
			return fmt.Errorf("bucket %s does not exist in project %s", s.config.Backup.GCP.Bucket, s.config.Backup.GCP.ProjectID)
		}
		// Autres types d'erreurs (permissions, réseau, etc.)
		s.log.WithFunc().WithError(err).WithField("bucket", s.config.Backup.GCP.Bucket).Error("Failed to access bucket")
		return fmt.Errorf("failed to access bucket %s: %w", s.config.Backup.GCP.Bucket, err)
	}

	// Log des informations du bucket si tout va bien
	s.log.WithFunc().WithFields(logrus.Fields{
		"bucket":   s.config.Backup.GCP.Bucket,
		"created":  attrs.Created,
		"location": attrs.Location,
		"project":  attrs.ProjectNumber,
	}).Info("Successfully connected to GCP bucket")

	// Tout est OK, on assigne le client
	s.gcsClient = client
	s.log.WithFunc().Info("GCP client initialized successfully")
	return nil
}

func (s *BackupService) Backup() error {
	s.log.WithFunc().Debug("Starting backup process")

	sourcePath := s.pathManager.GetBasePath()
	if _, err := os.Stat(sourcePath); err != nil {
		s.log.WithFunc().WithError(err).WithField("path", sourcePath).Error("Source path not accessible")
		return fmt.Errorf("source path not accessible: %w", err)
	}

	s.log.WithFunc().WithField("path", sourcePath).Debug("Starting backup from source path")

	if s.awsSession != nil {
		return s.backupToAWS(sourcePath)
	} else if s.gcsClient != nil {
		return s.backupToGCP(sourcePath)
	}
	return fmt.Errorf("no backup provider configured")

}

func (s *BackupService) backupToAWS(sourcePath string) error {
	s.log.WithFunc().WithFields(logrus.Fields{
		"source": sourcePath,
		"bucket": s.config.Backup.AWS.Bucket,
	}).Debug("Starting AWS backup")

	uploader := s3manager.NewUploader(s.awsSession)

	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.log.WithFunc().WithError(err).WithField("path", path).Error("Failed to access path")
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			s.log.WithFunc().WithError(err).WithField("path", path).Error("Failed to get relative path")
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			s.log.WithFunc().WithError(err).WithField("path", path).Error("Failed to open file")
			return err
		}
		defer file.Close()

		s.log.WithFunc().WithFields(logrus.Fields{
			"file":   relPath,
			"size":   info.Size(),
			"bucket": s.config.Backup.AWS.Bucket,
		}).Debug("Uploading file to AWS")

		_, err = uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(s.config.Backup.AWS.Bucket),
			Key:    aws.String(relPath),
			Body:   file,
		})

		if err != nil {
			s.log.WithFunc().WithError(err).WithField("file", relPath).Error("Failed to upload file")
			return fmt.Errorf("failed to upload %s: %w", relPath, err)
		}

		s.log.WithFunc().WithField("file", relPath).Info("File uploaded successfully")
		return nil
	})
}

func (s *BackupService) backupToGCP(sourcePath string) error {
	s.log.WithFunc().WithFields(logrus.Fields{
		"source": sourcePath,
		"bucket": s.config.Backup.GCP.Bucket,
	}).Debug("Starting GCP backup")

	ctx := context.Background()

	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.log.WithFunc().WithError(err).WithField("path", path).Error("Failed to access path")
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			s.log.WithFunc().WithError(err).WithField("path", path).Error("Failed to get relative path")
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			s.log.WithFunc().WithError(err).WithField("path", path).Error("Failed to open file")
			return err
		}
		defer file.Close()

		s.log.WithFunc().WithFields(logrus.Fields{
			"file":   relPath,
			"size":   info.Size(),
			"bucket": s.config.Backup.GCP.Bucket,
		}).Debug("Uploading file to GCP")

		obj := s.gcsClient.Bucket(s.config.Backup.GCP.Bucket).Object(relPath)
		writer := obj.NewWriter(ctx)

		if _, err := io.Copy(writer, file); err != nil {
			s.log.WithFunc().WithError(err).WithField("file", relPath).Error("Failed to upload file")
			return fmt.Errorf("failed to upload %s: %w", relPath, err)
		}

		if err := writer.Close(); err != nil {
			s.log.WithFunc().WithError(err).WithField("file", relPath).Error("Failed to finalize upload")
			return err
		}

		s.log.WithFunc().WithField("file", relPath).Info("File uploaded successfully")
		return nil
	})
}

func (s *BackupService) Restore() error {
	s.log.WithFunc().Debug("Starting restore process")

	if s.awsSession != nil {
		return s.restoreFromAWS()
	} else if s.gcsClient != nil {
		return s.restoreFromGCP()
	}

	return fmt.Errorf("no restore ")
}

func (s *BackupService) restoreFromGCP() error {
	s.log.WithFunc().Error("GCP restore not implemented")
	return fmt.Errorf("GCP restore not implemented")
}

func (s *BackupService) restoreFromAWS() error {
	s.log.WithFunc().Error("AWS restore not implemented")
	return fmt.Errorf("AWS restore not implemented")
}
