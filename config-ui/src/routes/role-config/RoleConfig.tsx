import { useEffect, useState } from 'react';
import { Button, Flex, Input, message, Popconfirm, Table } from 'antd';
import { PageHeader } from '@/components';
import API from '@/api';

export const RoleConfig = () => {
  const [roles, setRoles] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState('');

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await API.userconfig.getRoles();
      setRoles(res.roles || []);
    } catch {
      message.error('Failed to load roles');
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleNameChange = (roleId: string, value: string) => {
    setRoles((prev) => prev.map((r) => (r.id === roleId ? { ...r, Name: value } : r)));
  };

  const handleCreate = async () => {
    const name = newName.trim();
    if (!name) {
      message.error('Role name is required');
      return;
    }
    setCreating(true);
    try {
      await API.userconfig.createRole({ name });
      setNewName('');
      message.success('Role created');
      fetchData();
    } catch {
      message.error('Failed to create role');
    }
    setCreating(false);
  };

  const handleSave = async (role: any) => {
    const name = (role.Name || role.name || '').trim();
    if (!name) {
      message.error('Role name is required');
      return;
    }
    setSaving(role.id);
    try {
      await API.userconfig.updateRole(role.id, { name });
      message.success('Role updated');
      fetchData();
    } catch {
      message.error('Failed to update role');
    }
    setSaving(null);
  };

  const handleDelete = async (roleId: string) => {
    setSaving(roleId);
    try {
      await API.userconfig.deleteRole(roleId);
      message.success('Role deleted');
      fetchData();
    } catch {
      message.error('Failed to delete role');
    }
    setSaving(null);
  };

  return (
    <PageHeader
      breadcrumbs={[{ name: 'Role Config', path: '/role-config' }]}
      description="Create, rename, and remove roles. Role IDs are generated as github:Role:<n>."
    >
      <Flex gap={8} style={{ marginBottom: 16 }}>
        <Input
          placeholder="New role name"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          style={{ maxWidth: 360 }}
        />
        <Button type="primary" loading={creating} onClick={handleCreate}>
          Add Role
        </Button>
      </Flex>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={roles}
        columns={[
          {
            title: 'ID',
            dataIndex: 'id',
            width: 220,
          },
          {
            title: 'Name',
            render: (_, record) => (
              <Input
                value={record.Name || record.name || ''}
                onChange={(e) => handleNameChange(record.id, e.target.value)}
              />
            ),
          },
          {
            title: '',
            width: 220,
            render: (_, record) => (
              <Flex gap={8}>
                <Button
                  type="primary"
                  loading={saving === record.id}
                  onClick={() => handleSave(record)}
                >
                  Save
                </Button>
                <Popconfirm
                  title="Delete this role?"
                  onConfirm={() => handleDelete(record.id)}
                >
                  <Button danger loading={saving === record.id}>
                    Delete
                  </Button>
                </Popconfirm>
              </Flex>
            ),
          },
        ]}
      />
    </PageHeader>
  );
};
