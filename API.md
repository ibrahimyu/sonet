## API Reference

### Health Check

```
GET /api/health
```

Returns the health status and metadata about the service.

### Authentication

All requests must include a `X-User-ID` header with the user's ID. This ID is used to identify the user making the request.

### Posts

#### Create a Post

```
POST /api/posts
```

Request Body:
```json
{
  "content": "Hello world!",
  "image_url": "https://example.com/image.jpg",  // Optional
  "metadata": {                                  // Optional
    "location": "New York",
    "tags": ["hello", "world"]
  }
}
```

#### Get a Post

```
GET /api/posts/:id
```

#### List Posts

```
GET /api/posts
```

Query Parameters:
- `user_id` - Filter by user ID (optional)
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)

#### Search Posts

```
GET /api/posts/search
```

Query Parameters:
- `q` - Search query string (required)
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)

#### Update a Post

```
PUT /api/posts/:id
```

Request Body:
```json
{
  "content": "Updated content",
  "image_url": "https://example.com/new-image.jpg",
  "metadata": {
    "location": "Updated location"
  }
}
```

#### Delete a Post

```
DELETE /api/posts/:id
```

### Comments

#### Create a Comment

```
POST /api/comments
```

Request Body:
```json
{
  "post_id": "post-123",
  "content": "Great post!",
  "parent_id": "comment-456",  // Optional (for replies)
  "metadata": {}               // Optional
}
```

#### Get a Comment

```
GET /api/comments/:id
```

#### List Comments for a Post

```
GET /api/comments/post/:postId
```

Query Parameters:
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)

#### Update a Comment

```
PUT /api/comments/:id
```

Request Body:
```json
{
  "content": "Updated comment",
  "metadata": {}  // Optional
}
```

#### Delete a Comment

```
DELETE /api/comments/:id
```

### Reactions

#### Create/Toggle a Reaction

```
POST /api/reactions
```

Request Body:
```json
{
  "target_id": "post-123",
  "target_type": "post",       // "post" or "comment"
  "type": "like"               // Reaction type (like, love, haha, etc.)
}
```

#### List Reactions

```
GET /api/reactions/:targetType/:targetId
```

#### Delete a Reaction

```
DELETE /api/reactions/:id
```
