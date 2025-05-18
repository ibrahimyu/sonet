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
  "city": "New York",                           // Optional - City name
  "latitude": 40.7128,                          // Optional - Geographic coordinate
  "longitude": -74.0060,                        // Optional - Geographic coordinate
  "metadata": {                                  // Optional
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
- `q` - Search query string (optional if city or location provided)
- `city` - Filter by city name (optional)
- `lat` - Latitude coordinate for location-based search (optional, requires lng)
- `lng` - Longitude coordinate for location-based search (optional, requires lat)
- `radius` - Search radius in kilometers (default: 10, only used with lat/lng)
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)

At least one search parameter (q, city, or lat/lng) must be provided. When multiple parameters are used, they act as combined filters.

#### Update a Post

```
PUT /api/posts/:id
```

Request Body:
```json
{
  "content": "Updated content",
  "image_url": "https://example.com/new-image.jpg",
  "city": "Updated City",
  "latitude": 34.0522,
  "longitude": -118.2437,
  "metadata": {
    "tags": ["updated", "content"]
  }
}
```

#### Delete a Post

```
DELETE /api/posts/:id
```

#### List Posts by City

```
GET /api/posts/city/:cityName
```

Query Parameters:
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)

Returns posts from a specific city.

#### Find Nearby Posts

```
GET /api/posts/nearby
```

Query Parameters:
- `lat` - Latitude coordinate (required)
- `lng` - Longitude coordinate (required)
- `radius` - Search radius in kilometers (default: 10)
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)

Returns posts within the specified radius of the given coordinates.

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
