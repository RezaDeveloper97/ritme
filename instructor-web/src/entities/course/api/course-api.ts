import { api } from '@/shared/api';

import {
  chapterEnvelopeSchema,
  coursesSchema,
  dashboardSchema,
  lessonEnvelopeSchema,
  outlineSchema,
  type ChapterInput,
  type CourseInput,
  type LessonInput,
} from '../model/schema';

export const courseKeys = {
  all: ['courses'] as const,
  dashboard: () => [...courseKeys.all, 'dashboard'] as const,
  list: () => [...courseKeys.all, 'list'] as const,
  outline: (id: number) => [...courseKeys.all, 'outline', id] as const,
};

export const fetchDashboard = (signal?: AbortSignal) => api.get('/dashboard', { schema: dashboardSchema, signal });
export const fetchCourses = (signal?: AbortSignal) => api.get('/courses', { schema: coursesSchema, signal });
export const fetchOutline = (id: number, signal?: AbortSignal) =>
  api.get(`/courses/${id}`, { schema: outlineSchema, signal });

export const createCourse = (input: CourseInput) => api.post('/courses', input, { schema: outlineSchema });
export const updateCourse = (id: number, input: CourseInput) =>
  api.put(`/courses/${id}`, input, { schema: outlineSchema });
export const deleteCourse = (id: number) => api.delete(`/courses/${id}`);

export const createChapter = (courseId: number, input: ChapterInput) =>
  api.post(`/courses/${courseId}/chapters`, input, { schema: chapterEnvelopeSchema });
export const updateChapter = (courseId: number, id: number, input: ChapterInput) =>
  api.put(`/courses/${courseId}/chapters/${id}`, input, { schema: chapterEnvelopeSchema });
export const deleteChapter = (courseId: number, id: number) => api.delete(`/courses/${courseId}/chapters/${id}`);

export const createLesson = (courseId: number, input: LessonInput) =>
  api.post(`/courses/${courseId}/lessons`, input, { schema: lessonEnvelopeSchema });
export const updateLesson = (courseId: number, id: number, input: LessonInput) =>
  api.put(`/courses/${courseId}/lessons/${id}`, input, { schema: lessonEnvelopeSchema });
export const deleteLesson = (courseId: number, id: number) => api.delete(`/courses/${courseId}/lessons/${id}`);
