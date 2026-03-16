/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.
 */

import { request } from '@/utils';

export const getRepos = () =>
  request('/plugins/github/repos', {
    method: 'get',
  });

export const saveRepoTeamMapping = (data: { repo_id: string; team_id: string }) =>
  request('/plugins/github/repo-mapping', {
    method: 'post',
    data,
  });

export const uploadRepoTeamMapping = (file: File) => {
  const formData = new FormData();
  formData.append('file', file);
  return request('/plugins/github/repo-mapping/upload', {
    method: 'post',
    data: formData,
    headers: { 'Content-Type': 'multipart/form-data' },
  });
};