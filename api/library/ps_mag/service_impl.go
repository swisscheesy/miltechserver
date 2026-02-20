package ps_mag

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
)

const (
	PSMagContainerName = "library"
	PSMagPrefix        = "ps-mag/"
	PageSize           = 50
)

var issueRegex = regexp.MustCompile(`^PS_Magazine_Issue_(\d+)_([A-Za-z]+)_(\d{4})\.pdf$`)

type ServiceImpl struct {
	blobClient *azblob.Client
	credential *azblob.SharedKeyCredential
}

func NewService(blobClient *azblob.Client, credential *azblob.SharedKeyCredential) Service {
	return &ServiceImpl{
		blobClient: blobClient,
		credential: credential,
	}
}

// parseIssueFilename extracts issue metadata from a PS Magazine filename.
// Returns false if the name does not match the expected convention.
func parseIssueFilename(name string) (issueNumber int, month string, year int, ok bool) {
	matches := issueRegex.FindStringSubmatch(name)
	if matches == nil {
		return 0, "", 0, false
	}
	issueNumber, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, "", 0, false
	}
	month = matches[2]
	year, err = strconv.Atoi(matches[3])
	if err != nil {
		return 0, "", 0, false
	}
	return issueNumber, month, year, true
}

// filterByYear returns only issues matching the given year.
func filterByYear(issues []PSMagIssueResponse, year int) []PSMagIssueResponse {
	out := make([]PSMagIssueResponse, 0, len(issues))
	for _, issue := range issues {
		if issue.Year == year {
			out = append(out, issue)
		}
	}
	return out
}

// filterByIssueNumber returns only issues matching the given issue number.
func filterByIssueNumber(issues []PSMagIssueResponse, issueNumber int) []PSMagIssueResponse {
	out := make([]PSMagIssueResponse, 0, len(issues))
	for _, issue := range issues {
		if issue.IssueNumber == issueNumber {
			out = append(out, issue)
		}
	}
	return out
}

// sortIssues sorts issues in-place by IssueNumber. order must be "asc" or "desc".
func sortIssues(issues []PSMagIssueResponse, order string) {
	sort.Slice(issues, func(i, j int) bool {
		if order == "asc" {
			return issues[i].IssueNumber < issues[j].IssueNumber
		}
		return issues[i].IssueNumber > issues[j].IssueNumber
	})
}

// paginateIssues returns the page window and total page count.
// page is 1-indexed. Returns an empty slice (never nil) when page is beyond the end.
func paginateIssues(issues []PSMagIssueResponse, page, pageSize int) (pageItems []PSMagIssueResponse, totalPages int) {
	total := len(issues)
	totalPages = (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []PSMagIssueResponse{}, totalPages
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return issues[start:end], totalPages
}

// listAllIssues fetches every blob under ps-mag/ and parses metadata from filenames.
// Blobs that do not match the filename convention are silently skipped.
func (s *ServiceImpl) listAllIssues() ([]PSMagIssueResponse, error) {
	ctx := context.Background()
	containerClient := s.blobClient.ServiceClient().NewContainerClient(PSMagContainerName)
	prefix := PSMagPrefix
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	var issues []PSMagIssueResponse

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBlobListFailed, err)
		}
		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}
			blobPath := *blob.Name
			parts := strings.Split(blobPath, "/")
			fileName := parts[len(parts)-1]

			issueNum, month, year, ok := parseIssueFilename(fileName)
			if !ok {
				slog.Debug("Skipping non-matching ps-mag blob", "blobPath", blobPath)
				continue
			}

			var sizeBytes int64
			if blob.Properties != nil && blob.Properties.ContentLength != nil {
				sizeBytes = *blob.Properties.ContentLength
			}
			var lastModified string
			if blob.Properties != nil && blob.Properties.LastModified != nil {
				lastModified = blob.Properties.LastModified.Format(time.RFC3339)
			}

			issues = append(issues, PSMagIssueResponse{
				Name:         fileName,
				BlobPath:     blobPath,
				IssueNumber:  issueNum,
				Month:        month,
				Year:         year,
				SizeBytes:    sizeBytes,
				LastModified: lastModified,
			})
		}
	}
	return issues, nil
}

// ListIssues returns a paginated, optionally filtered list of PS Magazine issues.
func (s *ServiceImpl) ListIssues(page int, order string, year *int, issueNumber *int) (*PSMagIssuesResponse, error) {
	if page < 1 {
		return nil, ErrInvalidPage
	}
	order = strings.ToLower(order)
	if order != "asc" && order != "desc" {
		return nil, ErrInvalidOrder
	}

	issues, err := s.listAllIssues()
	if err != nil {
		return nil, err
	}

	if year != nil {
		issues = filterByYear(issues, *year)
	}
	if issueNumber != nil {
		issues = filterByIssueNumber(issues, *issueNumber)
	}

	sortIssues(issues, order)

	pageItems, totalPages := paginateIssues(issues, page, PageSize)

	return &PSMagIssuesResponse{
		Issues:     pageItems,
		Count:      len(pageItems),
		TotalCount: len(issues),
		Page:       page,
		TotalPages: totalPages,
		Order:      order,
	}, nil
}

// GenerateDownloadURL creates a 1-hour SAS URL for a ps-mag blob.
func (s *ServiceImpl) GenerateDownloadURL(blobPath string) (*DownloadURLResponse, error) {
	if strings.TrimSpace(blobPath) == "" {
		return nil, ErrEmptyBlobPath
	}
	if !strings.HasPrefix(blobPath, PSMagPrefix) {
		return nil, ErrInvalidBlobPath
	}
	if !strings.HasSuffix(strings.ToLower(blobPath), ".pdf") {
		return nil, ErrInvalidFileType
	}

	ctx := context.Background()
	blobClient := s.blobClient.ServiceClient().NewContainerClient(PSMagContainerName).NewBlobClient(blobPath)
	_, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		slog.Error("PS Magazine blob not found", "blobPath", blobPath, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrIssueNotFound, err)
	}

	expiryTime := time.Now().UTC().Add(1 * time.Hour)
	permissions := sas.BlobPermissions{Read: true}

	sasQueryParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     time.Now().UTC().Add(-5 * time.Minute),
		ExpiryTime:    expiryTime,
		Permissions:   permissions.String(),
		ContainerName: PSMagContainerName,
		BlobName:      blobPath,
	}.SignWithSharedKey(s.credential)
	if err != nil {
		slog.Error("Failed to generate SAS token for PS Magazine", "blobPath", blobPath, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrSASGenFailed, err)
	}

	downloadURL := fmt.Sprintf("%s?%s", blobClient.URL(), sasQueryParams.Encode())

	slog.Info("Generated PS Magazine download URL",
		"blobPath", blobPath,
		"expiresAt", expiryTime.Format(time.RFC3339))

	return &DownloadURLResponse{
		BlobPath:    blobPath,
		DownloadURL: downloadURL,
		ExpiresAt:   expiryTime.Format(time.RFC3339),
	}, nil
}
