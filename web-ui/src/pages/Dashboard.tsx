import { Card, CardContent, Grid, Typography } from '@mui/material'
import { useQuery } from '@tanstack/react-query'
import api from '@services/api'

export default function Dashboard() {
  const { data } = useQuery({
    queryKey: ['health'],
    queryFn: async () => (await api.get('/health')).data,
    refetchInterval: 10000,
  })

  return (
    <Grid container spacing={2}>
      <Grid item xs={12} md={4}>
        <Card>
          <CardContent>
            <Typography variant="h6">Server</Typography>
            <Typography>Status: {data?.server || 'unknown'}</Typography>
          </CardContent>
        </Card>
      </Grid>
      <Grid item xs={12} md={4}>
        <Card>
          <CardContent>
            <Typography variant="h6">RBAC</Typography>
            <Typography>Status: {data?.rbac?.status || 'unknown'}</Typography>
          </CardContent>
        </Card>
      </Grid>
      <Grid item xs={12} md={4}>
        <Card>
          <CardContent>
            <Typography variant="h6">Gadget Framework</Typography>
            <Typography>Status: {data?.gadget_framework || 'unknown'}</Typography>
          </CardContent>
        </Card>
      </Grid>
    </Grid>
  )
}


