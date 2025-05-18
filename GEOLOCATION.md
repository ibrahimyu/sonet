# Geolocation Features in Sonet

This document describes the geolocation features that have been added to the Sonet posting system.

## Overview

The Sonet posting system now supports geographical location data for posts. Users can include their city and precise geolocation (latitude and longitude) when creating posts. This allows for location-based queries to find posts that are from a specific city or within a certain distance from a geographical point.

## Data Model Changes

The `Post` model has been extended with the following fields:

- `City`: A string field storing the name of the city where the post was created
- `Latitude`: A float64 field storing the latitude coordinate
- `Longitude`: A float64 field storing the longitude coordinate

These fields are all optional but indexed for better query performance.

## New API Endpoints

The following new endpoints have been added:

### List Posts by City

```
GET /api/posts/city/:cityName
```

Returns posts from a specific city.

Query parameters:
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)

Example:
```
GET /api/posts/city/San%20Francisco
```

### Find Nearby Posts

```
GET /api/posts/nearby
```

Returns posts within a certain radius of a specified location.

Query parameters:
- `lat` - Latitude coordinate (required)
- `lng` - Longitude coordinate (required)
- `radius` - Search radius in kilometers (default: 10)
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)

Example:
```
GET /api/posts/nearby?lat=37.7749&lng=-122.4194&radius=30
```

### Enhanced Search

The existing search endpoint now supports location-based criteria:

```
GET /api/posts/search
```

Query parameters:
- `q` - Search query string (optional if city or location provided)
- `city` - Filter by city name (optional)
- `lat` - Latitude coordinate for location-based search (optional, requires lng)
- `lng` - Longitude coordinate for location-based search (optional, requires lat)
- `radius` - Search radius in kilometers (default: 10, only used with lat/lng)
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)

At least one search parameter (q, city, or lat/lng) must be provided. When multiple parameters are used, they act as combined filters.

Example:
```
GET /api/posts/search?q=Hello&city=Oakland
```

## Implementation Details

### SQLite Implementation

For SQLite, which doesn't have native geospatial functions, a two-step approach is used:

1. A bounding box query first filters posts that are roughly within the search area
2. Then, the Haversine formula is used to calculate precise distances and filter the final results

### PostgreSQL Implementation

For PostgreSQL, the implementation leverages the PostGIS extension for efficient geospatial queries:

1. The PostGIS extension is enabled during initialization
2. A spatial index is created on the location data
3. PostGIS functions like `ST_DWithin` and `ST_DistanceSphere` are used for efficient querying

## Performance Considerations

- Indexes have been added to the `City`, `Latitude`, and `Longitude` fields to improve query performance
- For PostgreSQL, a specialized spatial index has been added for optimized geospatial queries
- For SQLite, a bounding box approach is used to reduce the number of distance calculations needed

## Use Cases

These geolocation features enable several use cases:

1. **Hyperlocal Social Networks**: Users can see posts from people in their city
2. **Proximity-based Discovery**: Find posts that were created near the user's current location
3. **Travel & Tourism**: Users can discover posts from specific cities they're visiting
4. **Local Events & News**: Filter posts to see what's happening nearby
5. **Location-based Content**: Combine text search with location filters to find relevant nearby content
