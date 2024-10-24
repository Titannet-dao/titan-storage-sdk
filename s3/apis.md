
## Bucket APIs

### CreateBucket
- **Method**: `PUT`
- **HTTP Endpoint**: `PUT / HTTP/1.1 Host: <titan-bucket-url>`
- **Go SDK**: `func (s *storage) CreateBucket(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error)`

### DeleteBucket
- **Method**: `DELETE`
- **HTTP Endpoint**: `DELETE / HTTP/1.1 Host: <titan-bucket-url>`
- **Go SDK**: `func (s *storage) DeleteBucket(input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error)`

### ListBuckets
- **Method**: `GET`
- **HTTP Endpoint**: `GET / HTTP/1.1 Host: <titan-bucket-url>`
- **Go SDK**: `func (s *storage) ListBuckets(input *s3.ListBucketsInput) (*s3.ListBucketsOutput, error)`

### GetBucketLocation
- **Method**: `GET`
- **HTTP Endpoint**: `GET /?location HTTP/1.1 Host: <titan-bucket-url>`
- **Go SDK**: `func (s *storage) GetBucketLocation(input *s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error)`

## Object APIs

### PutObject
- **Description**: 上传一个对象到指定的存储桶。
- **Go SDK**: `func (s *storage) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)`

### GetObject
- **Description**: 从存储桶中检索对象。
- **Go SDK**: `func (s *storage) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)`

### DeleteObject
- **Method**: `DELETE`
- **HTTP Endpoint**: `DELETE /Key HTTP/1.1 Host: <titan-object-url>`
- **Go SDK**: `func (s *storage) DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)`

### ListObjects
- **Method**: `GET`
- **HTTP Endpoint**: `GET /?max-keys=2 HTTP/1.1 Host: <titan-object-url>`
- **Go SDK**: `func (s *storage) ListObjects(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error)`

### CopyObject
- **Method**: `PUT`
- **HTTP Endpoint**: `PUT /Key HTTP/1.1 Host: <titan-object-url>`
- **Go SDK**: `func (s *storage) CopyObject(input *s3.CopyObjectInput) (*s3.CopyObjectOutput, error)`

### HeadObject
- **Method**: `HEAD`
- **HTTP Endpoint**: `HEAD /Key HTTP/1.1 Host: <titan-object-url>`
- **Go SDK**: `func (s *storage) HeadObject(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error)`
