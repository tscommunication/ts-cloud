import { useEffect, useRef, useState } from 'react'
import { Alert, Box, CircularProgress } from '@mui/material'
import {
  importLibrary,
  setOptions,
} from '@googlemaps/js-api-loader'

interface GeoLocationPickerProps {
  latitude: number | null | undefined
  longitude: number | null | undefined
  onChange: (
    latitude: number,
    longitude: number,
  ) => void
  height?: number
}

const defaultCenter = {
  lat: 23.685,
  lng: 90.3563,
}

let loaderConfigured = false

function configureLoader() {
  if (loaderConfigured) {
    return
  }

  const apiKey =
    import.meta.env.VITE_GOOGLE_MAPS_API_KEY?.trim()

  if (!apiKey) {
    return
  }

  setOptions({
    key: apiKey,
    v: 'weekly',
  })

  loaderConfigured = true
}

export default function GeoLocationPicker({
  latitude,
  longitude,
  onChange,
  height = 360,
}: GeoLocationPickerProps) {
  const mapElementRef =
    useRef<HTMLDivElement | null>(null)

  const mapRef =
    useRef<google.maps.Map | null>(null)

  const markerRef =
    useRef<google.maps.marker.AdvancedMarkerElement | null>(
      null,
    )

  const latitudeRef = useRef(latitude)
  const longitudeRef = useRef(longitude)
  const onChangeRef = useRef(onChange)

  const apiKey =
    import.meta.env.VITE_GOOGLE_MAPS_API_KEY?.trim()

  const [loading, setLoading] =
    useState(Boolean(apiKey))
  const [error, setError] = useState('')

  const mapID =
    import.meta.env.VITE_GOOGLE_MAPS_MAP_ID?.trim()

  useEffect(() => {
    latitudeRef.current = latitude
  }, [latitude])

  useEffect(() => {
    longitudeRef.current = longitude
  }, [longitude])

  useEffect(() => {
    onChangeRef.current = onChange
  }, [onChange])

  useEffect(() => {
    if (!apiKey) {
      return
    }

    let cancelled = false

    const initialize = async () => {
      try {
        setLoading(true)
        setError('')

        configureLoader()

        const [
          { Map },
          { AdvancedMarkerElement },
        ] = await Promise.all([
          importLibrary('maps'),
          importLibrary('marker'),
        ])

        if (
          cancelled ||
          !mapElementRef.current
        ) {
          return
        }

        const initialLatitude =
          latitudeRef.current

        const initialLongitude =
          longitudeRef.current

        const hasCoordinates =
          initialLatitude != null &&
          initialLongitude != null

        const position = hasCoordinates
          ? {
              lat: initialLatitude,
              lng: initialLongitude,
            }
          : defaultCenter

        const map = new Map(
          mapElementRef.current,
          {
            center: position,
            zoom: hasCoordinates ? 18 : 7,
            mapId: mapID || 'DEMO_MAP_ID',
            mapTypeControl: true,
            streetViewControl: true,
            fullscreenControl: true,
          },
        )

        const marker =
          new AdvancedMarkerElement({
            map,
            position,
            gmpDraggable: true,
            title: 'Customer location',
          })

        marker.addListener(
          'dragend',
          () => {
            const markerPosition =
              marker.position

            if (!markerPosition) {
              return
            }

            const lat =
              typeof markerPosition.lat ===
              'function'
                ? markerPosition.lat()
                : markerPosition.lat

            const lng =
              typeof markerPosition.lng ===
              'function'
                ? markerPosition.lng()
                : markerPosition.lng

            onChangeRef.current(
              Number(lat),
              Number(lng),
            )
          },
        )

        map.addListener(
          'click',
          (
            event: google.maps.MapMouseEvent,
          ) => {
            if (!event.latLng) {
              return
            }

            const nextLatitude =
              event.latLng.lat()

            const nextLongitude =
              event.latLng.lng()

            marker.position = {
              lat: nextLatitude,
              lng: nextLongitude,
            }

            onChangeRef.current(
              nextLatitude,
              nextLongitude,
            )
          },
        )

        mapRef.current = map
        markerRef.current = marker
      } catch (mapError) {
        console.error(
          'Failed to initialize Google Maps.',
          mapError,
        )

        if (!cancelled) {
          setError(
            'Google Maps could not be loaded.',
          )
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void initialize()

    return () => {
      cancelled = true

      if (markerRef.current) {
        markerRef.current.map = null
        markerRef.current = null
      }

      mapRef.current = null
    }
  }, [apiKey, mapID])

  useEffect(() => {
    if (
      latitude == null ||
      longitude == null
    ) {
      return
    }

    const position = {
      lat: latitude,
      lng: longitude,
    }

    if (markerRef.current) {
      markerRef.current.position =
        position
    }

    if (mapRef.current) {
      mapRef.current.setCenter(
        position,
      )
    }
  }, [latitude, longitude])

  if (!apiKey) {
    return (
      <Alert severity="info">
        Google Maps is not configured yet.
        Latitude and longitude can still
        be captured or entered manually.
      </Alert>
    )
  }

  return (
    <Box
      sx={{
        position: 'relative',
        width: '100%',
        minHeight: height,
      }}
    >
      {error && (
        <Alert
          severity="error"
          sx={{ mb: 1 }}
        >
          {error}
        </Alert>
      )}

      {loading && (
        <Box
          sx={{
            position: 'absolute',
            inset: 0,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 2,
            bgcolor: 'background.paper',
          }}
        >
          <CircularProgress />
        </Box>
      )}

      <Box
        ref={mapElementRef}
        sx={{
          width: '100%',
          height,
          borderRadius: 1,
          overflow: 'hidden',
        }}
      />
    </Box>
  )
}
