import { useState, useEffect } from 'react';
import { UploadOutlined } from '@ant-design/icons';
import { Flex, Table, Select, Button, Upload, message } from 'antd';
import { PageHeader } from '@/components';
import API from '@/api';

export const TeamRepoConfig = () => {
  const [repos, setRepos] = useState<any[]>([]);
  const [teams, setTeams] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState<string | null>(null);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [reposRes, teamsRes] = await Promise.all([
        API.teamRepoConfig.getRepos(),
        API.userconfig.getTeams(),
      ]);

      setRepos(reposRes.repos || []);
      setTeams(teamsRes.teams || []);
    } catch (err) {
      message.error('Failed to load data');
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleChange = (repoId: string, value: string) => {
    setRepos((prev) =>
      prev.map((r) => (r.id === repoId ? { ...r, team_id: value } : r))
    );
  };

  const handleSave = async (repo: any) => {
    setSaving(repo.id);
    try {
      await API.teamRepoConfig.saveRepoTeamMapping({
        repo_id: repo.id,
        team_id: repo.team_id || '',
      });

      message.success('Repository mapping saved');
    } catch {
      message.error('Failed to save mapping');
    }
    setSaving(null);
  };

  const handleUpload = async (file: any) => {
    try {
      const res = await API.teamRepoConfig.uploadRepoTeamMapping(file);
      message.success(`Uploaded! ${res.saved} repositories mapped.`);
      fetchData();
    } catch {
      message.error('Upload failed');
    }
    return false;
  };

  return (
    <PageHeader
     breadcrumbs={[{ name: 'Team Repo Config', path: '/repo-team-config' }]}
     description="Map repositories to teams. Upload a CSV file with columns 'repo' and 'team' to automatically map repositories to teams, or manually assign a team to each repository."
    >
      <Flex justify="space-between" style={{ marginBottom: 16 }}>
        <Upload beforeUpload={handleUpload} showUploadList={false} accept=".csv">
          <Button icon={<UploadOutlined />}>Upload CSV</Button>
        </Upload>
      </Flex>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={repos}
        columns={[
          {
            title: 'Repository',
            dataIndex: 'name',
            render: (_, record) => (
              <div>
                <div style={{ fontWeight: 500 }}>
                  {record.repo_name || record.name || record.id}
                </div>
                {record.url && (
                  <div style={{ fontSize: 12, color: '#888' }}>
                    {record.url}
                  </div>
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
                onChange={(val) => handleChange(record.id, val)}
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