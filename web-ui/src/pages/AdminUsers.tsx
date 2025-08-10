import { useQuery } from '@tanstack/react-query'
import api from '@services/api'
import { DataGrid, GridColDef } from '@mui/x-data-grid'
import { Box, Paper, Typography } from '@mui/material'

export default function AdminUsers() {
  const { data } = useQuery({
    queryKey: ['rbac-users'],
    queryFn: async () => (await api.get('/api/rbac/users')).data,
  })

  const columns: GridColDef[] = [
    { field: 'username', headerName: 'Username', flex: 1 },
    { field: 'roles', headerName: 'Roles', flex: 1, valueGetter: params => (params.row.roles || []).join(', ') },
  ]

  const rows = (data?.users || []).map((u: any, id: number) => ({ id, ...u }))

  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="h6" sx={{ mb: 2 }}>Users</Typography>
      <Box sx={{ height: 400 }}>
        <DataGrid rows={rows} columns={columns} density="compact" disableRowSelectionOnClick />
      </Box>
    </Paper>
  )
}


