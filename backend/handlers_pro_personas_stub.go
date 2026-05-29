//go:build !windows

package main

import "net/http"

func handleStockAnalysis(w http.ResponseWriter, r *http.Request)  { proxyToPython(w, r, "/stock/analysis") }
func handleMedicalSearch(w http.ResponseWriter, r *http.Request)  { proxyToPython(w, r, "/medical/search") }
func handleContractReview(w http.ResponseWriter, r *http.Request) { proxyToPython(w, r, "/contract/review") }
func handleContentScript(w http.ResponseWriter, r *http.Request)  { proxyToPython(w, r, "/content/script") }
func handleLegalSearch(w http.ResponseWriter, r *http.Request)    { proxyToPython(w, r, "/legal/search") }
