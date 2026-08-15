'use client';

import Link from 'next/link';
import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';
import { useToast } from '@/components/ToastProvider';
import { buttonStyles } from '@/components/ui';
import { ApiError, NetworkError, adminTagApi } from '@/lib/api';
import type {
  AdminProblemDetail,
  AdminTag,
  CreateAdminProblemRequest,
  Difficulty,
  ProblemWriteExample,
  UpdateAdminProblemRequest,
} from '@/lib/types';
import { AdminApiError, AdminLoadingState, adminCard, adminField } from './AdminStates';

type ProblemFormValues = {
  title: string;
  slug: string;
  description: string;
  inputFormat: string;
  outputFormat: string;
  difficulty: Difficulty;
  tagIds: number[];
  examples: ProblemWriteExample[];
  constraintsText: string;
  hintsText: string;
  timeLimit: string;
  memoryLimit: string;
};

function errorMessage(err: unknown) {
  if (err instanceof NetworkError) return 'Cannot reach the API gateway.';
  if (err instanceof ApiError) return err.message || `Request failed with ${err.httpStatus}.`;
  return 'Request failed.';
}

function emptyExample(): ProblemWriteExample {
  return { input: '', expected_output: '', explanation: '' };
}

function fromProblem(problem?: AdminProblemDetail): ProblemFormValues {
  return {
    title: problem?.title ?? '',
    slug: problem?.slug ?? '',
    description: problem?.description ?? '',
    inputFormat: problem?.input_format ?? '',
    outputFormat: problem?.output_format ?? '',
    difficulty: problem?.difficulty ?? 'easy',
    tagIds: problem?.tags?.map((tag) => tag.id) ?? [],
    examples: problem?.examples?.length
      ? problem.examples.map((example) => ({
          input: example.input,
          expected_output: example.expected_output,
          explanation: example.explanation ?? '',
        }))
      : [emptyExample()],
    constraintsText: problem?.constraints?.join('\n') ?? '',
    hintsText: problem?.hints?.join('\n') ?? '',
    timeLimit: String(problem?.time_limit ?? 1),
    memoryLimit: String(problem?.memory_limit ?? 256),
  };
}

function splitLines(value: string) {
  return value
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean);
}

function buildCreate(values: ProblemFormValues): CreateAdminProblemRequest {
  return {
    title: values.title.trim(),
    description: values.description.trim(),
    input_format: values.inputFormat.trim(),
    output_format: values.outputFormat.trim(),
    difficulty: values.difficulty,
    tag_ids: values.tagIds,
    examples: values.examples.map((example) => ({
      input: example.input,
      expected_output: example.expected_output,
      explanation: example.explanation?.trim() || undefined,
    })),
    constraints: splitLines(values.constraintsText),
    hints: splitLines(values.hintsText),
    time_limit: Number(values.timeLimit),
    memory_limit: Number(values.memoryLimit),
  };
}

function buildUpdate(values: ProblemFormValues): UpdateAdminProblemRequest {
  return {
    ...buildCreate(values),
    slug: values.slug.trim(),
  };
}

function validate(values: ProblemFormValues) {
  if (values.title.trim().length < 3) return 'Title must be at least 3 characters.';
  if (values.description.trim().length < 3) return 'Description must be at least 3 characters.';
  if (!values.inputFormat.trim()) return 'Input format is required.';
  if (!values.outputFormat.trim()) return 'Output format is required.';
  if (Number(values.timeLimit) <= 0 || Number(values.timeLimit) > 30) return 'Time limit must be between 1 and 30 seconds.';
  if (Number(values.memoryLimit) < 16 || Number(values.memoryLimit) > 1024) return 'Memory limit must be between 16 and 1024 MB.';
  if (!values.examples.length) return 'At least one example is required.';
  if (values.examples.some((example) => !example.input.trim() || !example.expected_output.trim())) {
    return 'Every example needs input and expected output.';
  }
  return '';
}

export function AdminProblemForm({
  mode,
  problem,
  onSubmit,
  cancelHref = '/admin/problems',
}: {
  mode: 'create' | 'edit';
  problem?: AdminProblemDetail;
  onSubmit: (body: CreateAdminProblemRequest | UpdateAdminProblemRequest) => Promise<void>;
  cancelHref?: string;
}) {
  const { showToast } = useToast();
  const [values, setValues] = useState<ProblemFormValues>(() => fromProblem(problem));
  const [tags, setTags] = useState<AdminTag[]>([]);
  const [tagsLoading, setTagsLoading] = useState(true);
  const [tagsError, setTagsError] = useState('');
  const [saving, setSaving] = useState(false);
  const savingRef = useRef(false);

  const loadTags = useCallback((signal?: AbortSignal) => {
    setTagsLoading(true);
    adminTagApi
      .list(signal)
      .then((res) => {
        setTags(res.items ?? []);
        setTagsError('');
      })
      .catch((err) => {
        if (!signal?.aborted) setTagsError(errorMessage(err));
      })
      .finally(() => {
        if (!signal?.aborted) setTagsLoading(false);
      });
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    loadTags(controller.signal);
    return () => controller.abort();
  }, [loadTags]);

  const setValue = <K extends keyof ProblemFormValues>(key: K, value: ProblemFormValues[K]) => {
    setValues((current) => ({ ...current, [key]: value }));
  };

  const updateExample = (index: number, patch: Partial<ProblemWriteExample>) => {
    setValues((current) => ({
      ...current,
      examples: current.examples.map((example, currentIndex) => (currentIndex === index ? { ...example, ...patch } : example)),
    }));
  };

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (savingRef.current) return;
    const validation = validate(values);
    if (validation) {
      showToast(validation, 'error');
      return;
    }
    savingRef.current = true;
    setSaving(true);
    try {
      await onSubmit(mode === 'create' ? buildCreate(values) : buildUpdate(values));
    } catch (err) {
      showToast(errorMessage(err), 'error');
    } finally {
      setSaving(false);
      savingRef.current = false;
    }
  };

  return (
    <form onSubmit={submit} style={{ display: 'grid', gap: 14 }}>
      <section style={{ ...adminCard, padding: 16 }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 12 }}>
          <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
            Title
            <input
              required
              value={values.title}
              onChange={(event) => setValue('title', event.target.value)}
              className="ac-input"
              style={adminField}
            />
          </label>
          {mode === 'edit' && (
            <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
              Slug
              <input
                required
                value={values.slug}
                onChange={(event) => setValue('slug', event.target.value)}
                className="ac-input"
                style={adminField}
              />
            </label>
          )}
          <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
            Difficulty
            <select
              value={values.difficulty}
              onChange={(event) => setValue('difficulty', event.target.value as Difficulty)}
              className="ac-field"
              style={adminField}
            >
              <option value="easy">Easy</option>
              <option value="medium">Medium</option>
              <option value="hard">Hard</option>
            </select>
          </label>
          <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
            Time limit
            <input
              type="number"
              min={1}
              max={30}
              step={0.5}
              value={values.timeLimit}
              onChange={(event) => setValue('timeLimit', event.target.value)}
              className="ac-input"
              style={adminField}
            />
          </label>
          <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
            Memory limit
            <input
              type="number"
              min={16}
              max={1024}
              value={values.memoryLimit}
              onChange={(event) => setValue('memoryLimit', event.target.value)}
              className="ac-input"
              style={adminField}
            />
          </label>
        </div>
        <label style={{ display: 'grid', gap: 6, marginTop: 12, fontSize: 12, fontWeight: 650 }}>
          Description
          <span style={{ fontSize: 11.5, fontWeight: 400, color: 'var(--text3)' }}>Problem story and requirements for the reader.</span>
          <textarea
            required
            value={values.description}
            onChange={(event) => setValue('description', event.target.value)}
            className="ac-input"
            rows={7}
            style={{ ...adminField, height: 'auto', minHeight: 150, padding: 10, lineHeight: 1.55, resize: 'vertical' }}
          />
        </label>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: 12, marginTop: 12 }}>
          <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
            Input format
            <span style={{ fontSize: 11.5, fontWeight: 400, color: 'var(--text3)' }}>Explain the input structure in normal prose.</span>
            <textarea
              required
              value={values.inputFormat}
              onChange={(event) => setValue('inputFormat', event.target.value)}
              className="ac-input"
              rows={5}
              style={{ ...adminField, height: 'auto', minHeight: 110, padding: 10, lineHeight: 1.55, resize: 'vertical' }}
            />
          </label>
          <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
            Output format
            <span style={{ fontSize: 11.5, fontWeight: 400, color: 'var(--text3)' }}>Explain exactly what the program must print.</span>
            <textarea
              required
              value={values.outputFormat}
              onChange={(event) => setValue('outputFormat', event.target.value)}
              className="ac-input"
              rows={5}
              style={{ ...adminField, height: 'auto', minHeight: 110, padding: 10, lineHeight: 1.55, resize: 'vertical' }}
            />
          </label>
        </div>
      </section>

      <section style={{ ...adminCard, padding: 16 }}>
        <h2 style={{ margin: '0 0 10px', fontSize: 15 }}>Tags</h2>
        {tagsLoading ? (
          <AdminLoadingState title="Loading tags" />
        ) : tagsError ? (
          <AdminApiError title="Could not load tags" error={tagsError} onRetry={() => loadTags()} />
        ) : tags.length === 0 ? (
          <p style={{ margin: 0, color: 'var(--text2)', fontSize: 13 }}>No tags are available.</p>
        ) : (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {tags.map((tag) => {
              const selected = values.tagIds.includes(tag.id);
              return (
                <label
                  key={tag.id}
                  style={{
                    minHeight: 36,
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: 7,
                    padding: '0 10px',
                    borderRadius: 8,
                    border: `1px solid ${selected ? 'var(--accent-soft2)' : 'var(--border)'}`,
                    background: selected ? 'var(--accent-soft)' : 'var(--surface)',
                    color: tag.is_active ? 'var(--text)' : 'var(--text3)',
                    fontSize: 12.5,
                    cursor: 'pointer',
                  }}
                >
                  <input
                    type="checkbox"
                    checked={selected}
                    onChange={(event) => {
                      setValue(
                        'tagIds',
                        event.target.checked ? [...values.tagIds, tag.id] : values.tagIds.filter((id) => id !== tag.id),
                      );
                    }}
                  />
                  {tag.name}
                  {!tag.is_active && <span style={{ color: 'var(--warn)' }}>Inactive</span>}
                </label>
              );
            })}
          </div>
        )}
      </section>

      <section style={{ ...adminCard, padding: 16 }}>
        <h2 style={{ margin: '0 0 10px', fontSize: 15 }}>Examples</h2>
        <div style={{ display: 'grid', gap: 12 }}>
          {values.examples.map((example, index) => (
            <fieldset key={index} style={{ display: 'grid', gap: 10, margin: 0, padding: 12, border: '1px solid var(--border)', borderRadius: 8 }}>
              <legend style={{ padding: '0 6px', fontSize: 12.5, fontWeight: 650 }}>Example {index + 1}</legend>
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
                {values.examples.length > 1 && (
                  <button
                    type="button"
                    onClick={() => setValue('examples', values.examples.filter((_, currentIndex) => currentIndex !== index))}
                    className="ac-hover-surface2"
                    style={{ ...buttonStyles.secondary(30), color: 'var(--error)' }}
                  >
                    Remove
                  </button>
                )}
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 10 }}>
                <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
                  Input
                  <textarea
                    required
                    value={example.input}
                    onChange={(event) => updateExample(index, { input: event.target.value })}
                    className="ac-input"
                    rows={4}
                    spellCheck={false}
                    style={{ ...adminField, height: 'auto', minHeight: 96, padding: 10, fontFamily: 'var(--font-mono)', resize: 'vertical' }}
                  />
                </label>
                <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
                  Output
                  <textarea
                    required
                    value={example.expected_output}
                    onChange={(event) => updateExample(index, { expected_output: event.target.value })}
                    className="ac-input"
                    rows={4}
                    spellCheck={false}
                    style={{ ...adminField, height: 'auto', minHeight: 96, padding: 10, fontFamily: 'var(--font-mono)', resize: 'vertical' }}
                  />
                </label>
              </div>
              <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
                Explanation <span style={{ color: 'var(--text3)', fontWeight: 400 }}>(optional)</span>
                <textarea
                  value={example.explanation ?? ''}
                  onChange={(event) => updateExample(index, { explanation: event.target.value })}
                  className="ac-input"
                  rows={3}
                  style={{ ...adminField, height: 'auto', minHeight: 76, padding: 10, resize: 'vertical' }}
                />
              </label>
            </fieldset>
          ))}
        </div>
        <button
          type="button"
          onClick={() => setValue('examples', [...values.examples, emptyExample()])}
          className="ac-hover-surface2"
          style={{ ...buttonStyles.secondary(36), marginTop: 10 }}
        >
          Add example
        </button>
      </section>

      <section style={{ ...adminCard, padding: 16 }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: 12 }}>
          <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
            Constraints
            <textarea
              value={values.constraintsText}
              onChange={(event) => setValue('constraintsText', event.target.value)}
              className="ac-input"
              rows={7}
              style={{ ...adminField, height: 'auto', padding: 10, resize: 'vertical' }}
            />
          </label>
          <label style={{ display: 'grid', gap: 6, fontSize: 12, fontWeight: 650 }}>
            Hints
            <textarea
              value={values.hintsText}
              onChange={(event) => setValue('hintsText', event.target.value)}
              className="ac-input"
              rows={7}
              style={{ ...adminField, height: 'auto', padding: 10, resize: 'vertical' }}
            />
          </label>
        </div>
      </section>

      <div style={{ display: 'flex', flexWrap: 'wrap', justifyContent: 'flex-end', gap: 8 }}>
        <Link href={cancelHref} className="ac-hover-surface2" style={buttonStyles.secondary(38)}>
          Cancel
        </Link>
        <button type="submit" disabled={saving} aria-busy={saving} className="ac-hover-accent" style={buttonStyles.primary(38)}>
          {saving ? 'Saving...' : mode === 'create' ? 'Create problem' : 'Save changes'}
        </button>
      </div>
    </form>
  );
}
