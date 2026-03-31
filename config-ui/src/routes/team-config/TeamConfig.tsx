import { useEffect, useState } from 'react';
import { Button, Flex, Input, message, Popconfirm, Table } from 'antd';
import { PageHeader } from '@/components';
import API from '@/api';

export const TeamConfig = () => {
  const [teams, setTeams] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState('');

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await API.userconfig.getTeams();
      setTeams(res.teams || []);
    } catch {
      message.error('Failed to load teams');
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleNameChange = (teamId: string, value: string) => {
    setTeams((prev) => prev.map((t) => (t.id === teamId ? { ...t, Name: value } : t)));
  };

  const handleCreate = async () => {
    const name = newName.trim();
    if (!name) {
      message.error('Team name is required');
      return;
    }
    setCreating(true);
    try {
      await API.userconfig.createTeam({ name });
      setNewName('');
      message.success('Team created');
      fetchData();
    } catch {
      message.error('Failed to create team');
    }
    setCreating(false);
  };

  const handleSave = async (team: any) => {
    const name = (team.Name || team.name || '').trim();
    if (!name) {
      message.error('Team name is required');
      return;
    }
    setSaving(team.id);
    try {
      await API.userconfig.updateTeam(team.id, { name });
      message.success('Team updated');
      fetchData();
    } catch {
      message.error('Failed to update team');
    }
    setSaving(null);
  };

  const handleDelete = async (teamId: string) => {
    setSaving(teamId);
    try {
      await API.userconfig.deleteTeam(teamId);
      message.success('Team deleted');
      fetchData();
    } catch {
      message.error('Failed to delete team');
    }
    setSaving(null);
  };

  return (
    <PageHeader
      breadcrumbs={[{ name: 'Team Config', path: '/team-config' }]}
      description="Create, rename, and remove teams. Team IDs are generated as github:Team:<n>."
    >
      <Flex gap={8} style={{ marginBottom: 16 }}>
        <Input
          placeholder="New team name"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          style={{ maxWidth: 360 }}
        />
        <Button type="primary" loading={creating} onClick={handleCreate}>
          Add Team
        </Button>
      </Flex>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={teams}
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
                  title="Delete this team?"
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
