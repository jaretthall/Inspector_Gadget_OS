import { Box, Typography } from '@mui/material'
import { useQuery } from '@tanstack/react-query'
import api from '@services/api'

export default function Footer() {
  const { data } = useQuery({
    queryKey: ['health'],
    queryFn: async () => {
      const res = await api.get('/health')
      return res.data
    },
    staleTime: 60_000,
  })
  const version: string | undefined = data?.version

  return (
    <Box component="footer" sx={{ p: 2, textAlign: 'center', bgcolor: 'background.paper' }}>
      <Typography variant="body2" color="text.secondary">
        Inspector Gadget OS {version ? `v${version}` : ''}
      </Typography>
    </Box>
  )
}


