package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CVEData represents CVE information
type CVEData struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Published   string `json:"published"`
}

// CVEFeedResponse represents the response from CVE API
type CVEFeedResponse struct {
	CVEDataType         string    `json:"CVE_data_type"`
	CVEDataFormat       string    `json:"CVE_data_format"`
	CVEDataVersion      string    `json:"CVE_data_version"`
	CVEDataNumberOfCVEs string    `json:"CVE_data_numberOfCVEs"`
	CVEDataTimestamp    string    `json:"CVE_data_timestamp"`
	CVEItems            []CVEItem `json:"CVE_Items"`
}

// CVEItem represents a single CVE item
type CVEItem struct {
	CVE struct {
		DataType    string `json:"data_type"`
		DataFormat  string `json:"data_format"`
		DataVersion string `json:"data_version"`
		CVEDataMeta struct {
			ID       string `json:"ID"`
			ASSIGNER string `json:"ASSIGNER"`
		} `json:"CVE_data_meta"`
		Description struct {
			DescriptionData []struct {
				Lang  string `json:"lang"`
				Value string `json:"value"`
			} `json:"description_data"`
		} `json:"description"`
	} `json:"cve"`
	PublishedDate    string `json:"publishedDate"`
	LastModifiedDate string `json:"lastModifiedDate"`
}

// KBArticle represents a Microsoft KB article
type KBArticle struct {
	ID          string
	Title       string
	Published   string
	Description string
}

// CVEManager manages CVE and KB data
type CVEManager struct {
	cveList []string
	kbList  []string
	lastFetch time.Time
	cacheDuration time.Duration
}

// NewCVEManager creates a new CVE manager
func NewCVEManager() *CVEManager {
	return &CVEManager{
		cacheDuration: 24 * time.Hour, // Cache for 24 hours
	}
}

// FetchCVEData fetches recent CVEs from the NIST CVE API
func (cm *CVEManager) FetchCVEData() ([]string, error) {
	// Check cache
	if len(cm.cveList) > 0 && time.Since(cm.lastFetch) < cm.cacheDuration {
		return cm.cveList, nil
	}
	
	// Fetch from NIST CVE API (last 20 CVEs)
	url := "https://services.nvd.nist.gov/rest/json/cves/2.0?resultsPerPage=50&orderBy=publishedDate&sortOrder=desc"
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Get(url)
	if err != nil {
		// Fallback to mock data if API fails
		return cm.getMockCVEs(), nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return cm.getMockCVEs(), nil
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return cm.getMockCVEs(), nil
	}
	
	var feed CVEFeedResponse
	if err := json.Unmarshal(body, &feed); err != nil {
		return cm.getMockCVEs(), nil
	}
	
	cveList := make([]string, 0, len(feed.CVEItems))
	for _, item := range feed.CVEItems {
		if len(cveList) >= 50 {
			break
		}
		cveID := item.CVE.CVEDataMeta.ID
		if cveID != "" {
			cveList = append(cveList, cveID)
		}
	}
	
	if len(cveList) == 0 {
		return cm.getMockCVEs(), nil
	}
	
	cm.cveList = cveList
	cm.lastFetch = time.Now()
	return cveList, nil
}

// FetchKBData fetches recent Microsoft KB articles
func (cm *CVEManager) FetchKBData() ([]string, error) {
	// Check cache
	if len(cm.kbList) > 0 && time.Since(cm.lastFetch) < cm.cacheDuration {
		return cm.kbList, nil
	}
	
	// Microsoft doesn't have a public API for KB articles, so we'll use mock data
	// In a real implementation, you might scrape or use RSS feeds
	kbList := cm.getMockKBs()
	
	cm.kbList = kbList
	cm.lastFetch = time.Now()
	return kbList, nil
}

// getMockCVEs returns mock CVE data for fallback
func (cm *CVEManager) getMockCVEs() []string {
	return []string{
		"CVE-2024-0001", "CVE-2024-0002", "CVE-2024-0003", "CVE-2024-0004", "CVE-2024-0005",
		"CVE-2024-0006", "CVE-2024-0007", "CVE-2024-0008", "CVE-2024-0009", "CVE-2024-0010",
		"CVE-2024-0011", "CVE-2024-0012", "CVE-2024-0013", "CVE-2024-0014", "CVE-2024-0015",
		"CVE-2024-0016", "CVE-2024-0017", "CVE-2024-0018", "CVE-2024-0019", "CVE-2024-0020",
		"CVE-2024-0021", "CVE-2024-0022", "CVE-2024-0023", "CVE-2024-0024", "CVE-2024-0025",
		"CVE-2024-0026", "CVE-2024-0027", "CVE-2024-0028", "CVE-2024-0029", "CVE-2024-0030",
		"CVE-2024-0031", "CVE-2024-0032", "CVE-2024-0033", "CVE-2024-0034", "CVE-2024-0035",
		"CVE-2024-0036", "CVE-2024-0037", "CVE-2024-0038", "CVE-2024-0039", "CVE-2024-0040",
		"CVE-2024-0041", "CVE-2024-0042", "CVE-2024-0043", "CVE-2024-0044", "CVE-2024-0045",
		"CVE-2024-0046", "CVE-2024-0047", "CVE-2024-0048", "CVE-2024-0049", "CVE-2024-0050",
	}
}

// getMockKBs returns mock KB article numbers
func (cm *CVEManager) getMockKBs() []string {
	// Generate KB numbers in format KB######
	kbList := make([]string, 0, 50)
	for i := 1; i <= 50; i++ {
		kbList = append(kbList, fmt.Sprintf("KB%07d", 5000000+i))
	}
	return kbList
}

// GetCVEList returns the cached CVE list
func (cm *CVEManager) GetCVEList() []string {
	if len(cm.cveList) == 0 {
		cves, _ := cm.FetchCVEData()
		return cves
	}
	return cm.cveList
}

// GetKBList returns the cached KB list
func (cm *CVEManager) GetKBList() []string {
	if len(cm.kbList) == 0 {
		kbs, _ := cm.FetchKBData()
		return kbs
	}
	return cm.kbList
}

// FormatCVEID formats a CVE ID for display (truncates if too long)
func FormatCVEID(cveID string, maxLen int) string {
	if len(cveID) <= maxLen {
		return cveID
	}
	// Try to keep the year and number visible
	parts := strings.Split(cveID, "-")
	if len(parts) >= 3 {
		// CVE-YYYY-NNNN format
		return parts[0] + "-" + parts[1] + "-" + parts[2][:maxLen-len(parts[0])-len(parts[1])-2]
	}
	return cveID[:maxLen]
}

// FormatKBID formats a KB ID for display
func FormatKBID(kbID string, maxLen int) string {
	if len(kbID) <= maxLen {
		return kbID
	}
	return kbID[:maxLen]
}

// "Now this is not the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942

