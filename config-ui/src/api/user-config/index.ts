/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.
 */

import { request } from '@/utils';

export const getUsers = () =>
  request('/plugins/github/users', {
    method: 'get',
  });

export const getTeams = () =>
  request('/plugins/github/teams', {
    method: 'get',
  });

export const getRoles = () =>
  request('/plugins/github/roles', {
    method: 'get',
  });

export const saveUserMapping = (data: { user_id: string; team_id: string; role_id: string }) =>
  request('/plugins/github/user-mapping', {
    method: 'post',
    data,
  });

  export const uploadUserMapping = (file: File) => {
  const formData = new FormData();
  formData.append('file', file);
  return request('/plugins/github/user-mapping/upload', {
    method: 'post',
    data: formData,
    headers: { 'Content-Type': 'multipart/form-data' },
  });
};