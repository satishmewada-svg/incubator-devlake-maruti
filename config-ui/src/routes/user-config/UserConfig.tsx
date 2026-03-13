import { useState, useEffect } from 'react';
import { UploadOutlined } from '@ant-design/icons';
import { Flex, Table, Select, Button, Upload, message } from 'antd';
import { PageHeader } from '@/components';
import API from '@/api';

export const UserConfig = () => {
  const [users, setUsers] = useState<any[]>([]);
  const [teams, setTeams] = useState<any[]>([]);
  const [roles, setRoles] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState<string | null>(null);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [usersRes, teamsRes, rolesRes] = await Promise.all([
        API.userconfig.getUsers(),
        API.userconfig.getTeams(),
        API.userconfig.getRoles(),
      ]);
      setUsers(usersRes.users || []);
      setTeams(teamsRes.teams || []);
      setRoles(rolesRes.roles || []);
    } catch (err) {
      message.error('Failed to load data');
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleChange = (userId: string, field: 'team_id' | 'role_id', value: string) => {
    setUsers((prev) =>
      prev.map((u) => (u.id === userId ? { ...u, [field]: value } : u))
    );
  };

  const handleSave = async (user: any) => {
    setSaving(user.id);
    try {
      await API.userconfig.saveUserMapping({
        user_id: user.id,
        team_id: user.team_id || '',
        role_id: user.role_id || '',
      });
      message.success('User mapping saved');
    } catch {
      message.error('Failed to save mapping');
    }
    setSaving(null);
  };

const handleUpload = async (file: any) => {
  try {
    const res = await API.userconfig.uploadUserMapping(file);
    message.success(`Uploaded! ${res.saved} users mapped, ${res.skipped} skipped.`);
    fetchData(); // refresh user list with new team/role
  } catch {
    message.error('Upload failed');
  }
  return false;
};
  return (
    <PageHeader
      breadcrumbs={[{ name: 'User Config', path: '/user-config' }]}
      description="Manage user team and role assignments. Upload a CSV file to automatically create teams and roles and map them to users, or manually assign teams and roles to individual users. Users are synced directly from GitHub repositories."
    >
      <Flex justify="space-between" style={{ marginBottom: 16 }}>
        <Upload beforeUpload={handleUpload} showUploadList={false} accept=".csv">
          <Button icon={<UploadOutlined />}>Upload CSV</Button>
        </Upload>
      </Flex>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={users}
        columns={[
          {
            title: 'User',
            dataIndex: 'name',
            render: (_, record) => (
              <div>
                <div style={{ fontWeight: 500 }}>
                  {record.user_full_name || record.name || record.id}
                </div>
                {record.email && (
                  <div style={{ fontSize: 12, color: '#888' }}>{record.email}</div>
                )}
              </div>
            ),
          },
          {
            title: 'Team',
            render: (_, record) => (
              <Select
                style={{ width: 220 }}
                placeholder="Select Team"
                value={record.team_id || undefined}
                options={teams.map((t) => ({
                  label: t.Name,
                  value: t.id,
                }))}
                onChange={(val) => handleChange(record.id, 'team_id', val)}
              />
            ),
          },
          {
            title: 'Role',
            render: (_, record) => (
              <Select
                style={{ width: 220 }}
                placeholder="Select Role"
                value={record.role_id || undefined}
                options={roles.map((r) => ({
                  label: r.Name,
                  value: r.id,
                }))}
                onChange={(val) => handleChange(record.id, 'role_id', val)}
              />
            ),
          },
          {
            title: '',
            render: (_, record) => (
              <Button
                type="primary"
                loading={saving === record.id}
                onClick={() => handleSave(record)}
              >
                Save
              </Button>
            ),
          },
        ]}
      />
    </PageHeader>
  );
};