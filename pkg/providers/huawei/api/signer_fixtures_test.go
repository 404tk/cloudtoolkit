package api

type signerFixture struct {
	Name     string
	Input    signerFixtureInput
	Expected signerFixtureExpected
}

type signerFixtureInput struct {
	Method       string
	Host         string
	Path         string
	Query        string
	XSdkDate     string
	AccessKey    string
	SecretKey    string
	ContentType  string
	Body         string
	ExtraHeaders map[string]string
}

type signerFixtureExpected struct {
	SignedHeaders    string
	CanonicalRequest string
	StringToSign     string
	Authorization    string
}

func signerFixtures() []signerFixture {
	return []signerFixture{
		{
			Name: "iam_list_regions",
			Input: signerFixtureInput{
				Method:    "GET",
				Host:      "iam.cn-north-4.myhuaweicloud.com",
				Path:      "/v3/regions",
				Query:     "",
				XSdkDate:  "20260419T120000Z",
				AccessKey: "HWIsigcaptureAKID",
				SecretKey: "HWIsigcaptureSECRET1234567890abcdefg",
			},
			Expected: signerFixtureExpected{
				SignedHeaders:    "x-sdk-date",
				CanonicalRequest: "GET\n/v3/regions/\n\nx-sdk-date:20260419T120000Z\n\nx-sdk-date\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "SDK-HMAC-SHA256\n20260419T120000Z\nc048f41d5b6fdf228624b8556815442a93e3d61eef13972f41ce6890bfed570c",
				Authorization:    "SDK-HMAC-SHA256 Access=HWIsigcaptureAKID, SignedHeaders=x-sdk-date, Signature=af3d1702fa512cef934a64555d74525b713f438d395c6626c51cba3554ffd5d7",
			},
		},
		{
			Name: "iam_show_permanent_access_key",
			Input: signerFixtureInput{
				Method:    "GET",
				Host:      "iam.cn-north-4.myhuaweicloud.com",
				Path:      "/v3.0/OS-CREDENTIAL/credentials/HWIsigcaptureAKID",
				Query:     "",
				XSdkDate:  "20260419T120000Z",
				AccessKey: "HWIsigcaptureAKID",
				SecretKey: "HWIsigcaptureSECRET1234567890abcdefg",
			},
			Expected: signerFixtureExpected{
				SignedHeaders:    "x-sdk-date",
				CanonicalRequest: "GET\n/v3.0/OS-CREDENTIAL/credentials/HWIsigcaptureAKID/\n\nx-sdk-date:20260419T120000Z\n\nx-sdk-date\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "SDK-HMAC-SHA256\n20260419T120000Z\nbb0406f50104e71972cec3b6e43572c263cef4b00ade699ece2f14c19d27e84b",
				Authorization:    "SDK-HMAC-SHA256 Access=HWIsigcaptureAKID, SignedHeaders=x-sdk-date, Signature=8c88d9fc5b2140b103487b11e35ce47336b5494292697819bfb9e4eda3335171",
			},
		},
		{
			Name: "iam_show_user",
			Input: signerFixtureInput{
				Method:    "GET",
				Host:      "iam.cn-north-4.myhuaweicloud.com",
				Path:      "/v3/users/fakeuserid0001",
				Query:     "",
				XSdkDate:  "20260419T120000Z",
				AccessKey: "HWIsigcaptureAKID",
				SecretKey: "HWIsigcaptureSECRET1234567890abcdefg",
			},
			Expected: signerFixtureExpected{
				SignedHeaders:    "x-sdk-date",
				CanonicalRequest: "GET\n/v3/users/fakeuserid0001/\n\nx-sdk-date:20260419T120000Z\n\nx-sdk-date\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "SDK-HMAC-SHA256\n20260419T120000Z\n885f19659dcd83924759f1e8fada5bc9800ffdb619a8969ede4a7fb08c2b30c6",
				Authorization:    "SDK-HMAC-SHA256 Access=HWIsigcaptureAKID, SignedHeaders=x-sdk-date, Signature=a50a7d97b2d322538193c840be34e85f9306c82a22ba7f8a6255bd4248a93ec3",
			},
		},
		{
			Name: "iam_list_users",
			Input: signerFixtureInput{
				Method:    "GET",
				Host:      "iam.cn-north-4.myhuaweicloud.com",
				Path:      "/v3/users",
				Query:     "",
				XSdkDate:  "20260419T120000Z",
				AccessKey: "HWIsigcaptureAKID",
				SecretKey: "HWIsigcaptureSECRET1234567890abcdefg",
			},
			Expected: signerFixtureExpected{
				SignedHeaders:    "x-sdk-date",
				CanonicalRequest: "GET\n/v3/users/\n\nx-sdk-date:20260419T120000Z\n\nx-sdk-date\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "SDK-HMAC-SHA256\n20260419T120000Z\n8c82f5d1588572224114e60d581ab29f899bba80de802a13b50540bb46eaa040",
				Authorization:    "SDK-HMAC-SHA256 Access=HWIsigcaptureAKID, SignedHeaders=x-sdk-date, Signature=a85ff09cfc816a2624a5fbfb152af654ab7bd64f1768ce351c120ebafa4c84fa",
			},
		},
		{
			Name: "iam_create_user",
			Input: signerFixtureInput{
				Method:      "POST",
				Host:        "iam.cn-north-4.myhuaweicloud.com",
				Path:        "/v3/users",
				Query:       "",
				XSdkDate:    "20260419T120000Z",
				AccessKey:   "HWIsigcaptureAKID",
				SecretKey:   "HWIsigcaptureSECRET1234567890abcdefg",
				ContentType: "application/json;charset=UTF-8",
				Body:        "{\"user\":{\"name\":\"ctk\",\"password\":\"P@ssw0rd!capture\",\"enabled\":true,\"domain_id\":\"fakedomainid0001\"}}",
				ExtraHeaders: map[string]string{
					"X-Domain-Id": "fakedomainid0001",
				},
			},
			Expected: signerFixtureExpected{
				SignedHeaders:    "x-domain-id;x-sdk-date",
				CanonicalRequest: "POST\n/v3/users/\n\nx-domain-id:fakedomainid0001\nx-sdk-date:20260419T120000Z\n\nx-domain-id;x-sdk-date\n5e0184eb7e4e041f67634ed8d3105bca050cb88b06fe576f63229ace4b192783",
				StringToSign:     "SDK-HMAC-SHA256\n20260419T120000Z\n4f38d1bac1a8e450536030de76b7ef2f62ce81bdc9c90c02987ab9d67bf4befc",
				Authorization:    "SDK-HMAC-SHA256 Access=HWIsigcaptureAKID, SignedHeaders=x-domain-id;x-sdk-date, Signature=3a60f134909cad3eb780a394f29c58345bbe46902a52078b2133b36d4dc8f8a8",
			},
		},
		{
			Name: "iam_list_groups",
			Input: signerFixtureInput{
				Method:    "GET",
				Host:      "iam.cn-north-4.myhuaweicloud.com",
				Path:      "/v3/groups",
				Query:     "",
				XSdkDate:  "20260419T120000Z",
				AccessKey: "HWIsigcaptureAKID",
				SecretKey: "HWIsigcaptureSECRET1234567890abcdefg",
				ExtraHeaders: map[string]string{
					"X-Domain-Id": "fakedomainid0001",
				},
			},
			Expected: signerFixtureExpected{
				SignedHeaders:    "x-domain-id;x-sdk-date",
				CanonicalRequest: "GET\n/v3/groups/\n\nx-domain-id:fakedomainid0001\nx-sdk-date:20260419T120000Z\n\nx-domain-id;x-sdk-date\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "SDK-HMAC-SHA256\n20260419T120000Z\n4c8599989dcf14464ea3d2f35d6fe5614836ece8e0dafc37a261d5701b3ccda3",
				Authorization:    "SDK-HMAC-SHA256 Access=HWIsigcaptureAKID, SignedHeaders=x-domain-id;x-sdk-date, Signature=9c5dabf62bb59aff34643e67ce23f8c801d1ecb84dc4ef23c7076228ce0b0f0b",
			},
		},
		{
			Name: "iam_add_user_to_group",
			Input: signerFixtureInput{
				Method:    "PUT",
				Host:      "iam.cn-north-4.myhuaweicloud.com",
				Path:      "/v3/groups/fakegroupid0001/users/fakeuserid0001",
				Query:     "",
				XSdkDate:  "20260419T120000Z",
				AccessKey: "HWIsigcaptureAKID",
				SecretKey: "HWIsigcaptureSECRET1234567890abcdefg",
				ExtraHeaders: map[string]string{
					"X-Domain-Id": "fakedomainid0001",
				},
			},
			Expected: signerFixtureExpected{
				SignedHeaders:    "x-domain-id;x-sdk-date",
				CanonicalRequest: "PUT\n/v3/groups/fakegroupid0001/users/fakeuserid0001/\n\nx-domain-id:fakedomainid0001\nx-sdk-date:20260419T120000Z\n\nx-domain-id;x-sdk-date\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "SDK-HMAC-SHA256\n20260419T120000Z\n456fcced25bdb7f14edefcd120ad0a5d1b1d6acf164f52d3c4e34beeefc1377b",
				Authorization:    "SDK-HMAC-SHA256 Access=HWIsigcaptureAKID, SignedHeaders=x-domain-id;x-sdk-date, Signature=23a423e2c8e61fbb4ab5a21ac5ab879f0f29896ce5c6dcd1c15c3efa883c783e",
			},
		},
		{
			Name: "iam_list_auth_domains",
			Input: signerFixtureInput{
				Method:    "GET",
				Host:      "iam.cn-north-4.myhuaweicloud.com",
				Path:      "/v3/auth/domains",
				Query:     "",
				XSdkDate:  "20260419T120000Z",
				AccessKey: "HWIsigcaptureAKID",
				SecretKey: "HWIsigcaptureSECRET1234567890abcdefg",
			},
			Expected: signerFixtureExpected{
				SignedHeaders:    "x-sdk-date",
				CanonicalRequest: "GET\n/v3/auth/domains/\n\nx-sdk-date:20260419T120000Z\n\nx-sdk-date\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "SDK-HMAC-SHA256\n20260419T120000Z\n1e702dc31ede47229f6f1a5260b725eee508534aa37211e5629303cc2e1bbece",
				Authorization:    "SDK-HMAC-SHA256 Access=HWIsigcaptureAKID, SignedHeaders=x-sdk-date, Signature=6fd0d42703b11866750d106833505d626948950567990df69b7a6a3e0a598dd3",
			},
		},
		{
			Name: "iam_delete_user",
			Input: signerFixtureInput{
				Method:    "DELETE",
				Host:      "iam.cn-north-4.myhuaweicloud.com",
				Path:      "/v3/users/fakeuserid0001",
				Query:     "",
				XSdkDate:  "20260419T120000Z",
				AccessKey: "HWIsigcaptureAKID",
				SecretKey: "HWIsigcaptureSECRET1234567890abcdefg",
				ExtraHeaders: map[string]string{
					"X-Domain-Id": "fakedomainid0001",
				},
			},
			Expected: signerFixtureExpected{
				SignedHeaders:    "x-domain-id;x-sdk-date",
				CanonicalRequest: "DELETE\n/v3/users/fakeuserid0001/\n\nx-domain-id:fakedomainid0001\nx-sdk-date:20260419T120000Z\n\nx-domain-id;x-sdk-date\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "SDK-HMAC-SHA256\n20260419T120000Z\naedd8ca85caca2f69f2cd0f3da875462b745078767821d5f31e26d3f987bbc67",
				Authorization:    "SDK-HMAC-SHA256 Access=HWIsigcaptureAKID, SignedHeaders=x-domain-id;x-sdk-date, Signature=71540f639b1310976bc95cfb9a4535b0ca8f8e561fd3540c8b49117a3f89eaab",
			},
		},
	}
}
