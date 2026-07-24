// SPDX-License-Identifier: Apache-2.0

package domain

import "time"

type ContentID int
type VersionID string

type Content struct {
	ContentID       ContentID  `json:"contentId"`
	Publisher       string     `json:"publisher"`
	Type            string     `json:"type"`
	Categories      []Category `json:"categories"`
	SupportCategory string     `json:"supportCategory"`
	LicenseURL      string     `json:"licenseUrl,omitempty"`
	IsPrivate       bool       `json:"isPrivate"`
	IsCompanyApproved bool     `json:"isCompanyApproved,omitempty"`
	LatestVersion   *Version   `json:"latestVersion,omitempty"`
}

type Category struct {
	Name string `json:"name"`
}

type Version struct {
	Name                      string    `json:"name"`
	VersionID                 string    `json:"versionId"`
	VersionNumber             string    `json:"versionNumber"`
	MinSupportedMendixVersion string    `json:"minSupportedMendixVersion"`
	PublicationDate           time.Time `json:"publicationDate"`
	ReleaseNotes              string    `json:"releaseNotes,omitempty"`
	VersionType               string    `json:"versionType,omitempty"`
	DownloadURL               string    `json:"downloadUrl,omitempty"`
}

type InstalledModule struct {
	Name            string
	ModuleID        string
	AppStoreGuid    string
	AppStoreVersion string
}
