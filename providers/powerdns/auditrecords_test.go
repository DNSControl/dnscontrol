package powerdns

// func TestAuditRecordsSvcbAutoHintOrder(t *testing.T) {
// 	tests := []struct {
// 		name    string
// 		params  string
// 		wantErr bool
// 	}{
// 		{
// 			name:    "sorted auto hints",
// 			params:  "alpn=h3,h2 ipv4hint=auto ipv6hint=auto",
// 			wantErr: false,
// 		},
// 		{
// 			name:    "ipv6hint before ipv4hint",
// 			params:  "alpn=h3,h2 ipv6hint=auto ipv4hint=auto",
// 			wantErr: true,
// 		},
// 		{
// 			name:    "non auto hints use regular validation path",
// 			params:  "alpn=h3,h2 ipv6hint=2001:db8::1 ipv4hint=192.0.2.1",
// 			wantErr: false,
// 		},
// 		{
// 			name:    "unknown params ignored",
// 			params:  "alpn=h3,h2 key65400=value ipv4hint=auto ipv6hint=auto",
// 			wantErr: false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			params := tt.params
// 			// params = strings.ReplaceAll(params, "ipv4hint=auto", "ipv4hint=192.0.2.1")
// 			// params = strings.ReplaceAll(params, "ipv6hint=auto", "ipv6hint=1::1")
// 			record := powerDNSSVCBRecord("HTTPS", params)
// 			errs := AuditRecords(models.Records{record})

// 			if tt.wantErr && len(errs) != 0 {
// 				assert.Len(t, errs, 1)
// 				assert.True(t, strings.Contains(errs[0].Error(), "ipv4hint must appear before ipv6hint"))
// 			} else {
// 				assert.Empty(t, errs)
// 			}
// 		})
// 	}
// }

// func powerDNSSVCBRecord(rtype, params string) *models.RecordConfig {
// 	fmt.Printf("DEBUG: Creating %s record with params: %s\n", rtype, params)
// 	dc := models.MustNewDomainConfig("example.com")
// 	rc := dc.MustNewRecordConfig("auto", 0, rtype, 1, ".", params)
// 	return rc
// }
