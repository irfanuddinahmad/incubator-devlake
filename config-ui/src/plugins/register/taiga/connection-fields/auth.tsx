/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

import { useEffect } from 'react';
import { Input } from 'antd';

interface Props {
  type: 'create' | 'update';
  initialValues: any;
  values: any;
  errors: any;
  setValues: (value: any) => void;
  setErrors: (value: any) => void;
}

export const Auth = ({ type, initialValues, values, setValues, setErrors }: Props) => {
  useEffect(() => {
    setValues({
      endpoint: initialValues.endpoint,
      username: initialValues.username,
      password: initialValues.password,
    });
  }, [initialValues.endpoint, initialValues.username, initialValues.password]);

  useEffect(() => {
    const required = (values.username && values.password) || type === 'update';
    setErrors({
      endpoint: !values.endpoint ? 'endpoint is required' : '',
      auth: required ? '' : 'auth is required',
    });
  }, [values]);

  const handleChangeEndpoint = (e: React.ChangeEvent<HTMLInputElement>) => {
    setValues({
      endpoint: e.target.value,
    });
  };

  const handleChangeUsername = (e: React.ChangeEvent<HTMLInputElement>) => {
    setValues({
      username: e.target.value,
    });
  };

  const handleChangePassword = (e: React.ChangeEvent<HTMLInputElement>) => {
    setValues({
      password: e.target.value,
    });
  };

  return (
    <>
      <div className="form-item">
        <label>
          <span className="label">Endpoint URL</span>
          <span className="required">*</span>
        </label>
        <Input
          style={{ width: 386 }}
          placeholder="https://projects.example.com/api/v1/"
          value={values.endpoint}
          onChange={handleChangeEndpoint}
        />
        <p className="description">
          Provide the Taiga instance API URL. For Taiga Cloud, use https://api.taiga.io/api/v1/
        </p>
      </div>
      <div className="form-item">
        <label>
          <span className="label">Username</span>
          <span className="required">*</span>
        </label>
        <Input
          style={{ width: 386 }}
          placeholder="username or email"
          value={values.username}
          onChange={handleChangeUsername}
        />
      </div>
      <div className="form-item">
        <label>
          <span className="label">Password</span>
          <span className="required">*</span>
        </label>
        <Input.Password
          style={{ width: 386 }}
          placeholder={type === 'update' ? '********' : 'password'}
          value={values.password}
          onChange={handleChangePassword}
        />
      </div>
    </>
  );
};
