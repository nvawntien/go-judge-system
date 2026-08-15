'use client';

import Link from 'next/link';
import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';
import { useToast } from '@/components/ToastProvider';
import { buttonStyles } from '@/components/ui';
import { AdminIcon } from './AdminIcons';
import { ApiError, NetworkError, problemApi } from '@/lib/api';
import type {
  AdminProblemDetail,
  CreateAdminProblemRequest,
  Difficulty,
  ProblemWriteExample,
  Tag,
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
  constraints: string[];
  hints: string[];
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
    constraints: problem?.constraints ?? [],
    hints: problem?.hints ?? [],
    timeLimit: String(problem?.time_limit ?? 1),
    memoryLimit: String(problem?.memory_limit ?? 256),
  };
}

function normalizeItems(items: string[]) {
  return items.map((item) => item.trim());
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
    constraints: normalizeItems(values.constraints),
    hints: normalizeItems(values.hints),
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
  if (values.constraints.some((constraint) => !constraint.trim())) {
    return 'Complete or remove empty constraints before saving.';
  }
  if (values.hints.some((hint) => !hint.trim())) {
    return 'Complete or remove empty hints before saving.';
  }
  return '';
}

function StringListField({
  label,
  description,
  items,
  onChange,
  multiline = false,
}: {
  label: 'Constraints' | 'Hints';
  description: string;
  items: string[];
  onChange: (items: string[]) => void;
  multiline?: boolean;
}) {
  const itemName = label === 'Constraints' ? 'constraint' : 'hint';

  const updateItem = (index: number, value: string) => {
    onChange(items.map((item, currentIndex) => (currentIndex === index ? value : item)));
  };

  return (
    <section style={{ ...adminCard, padding: 16 }}>
      <div style={{ display: 'grid', gap: 3, marginBottom: items.length ? 12 : 10 }}>
        <h2 style={{ margin: 0, fontSize: 15 }}>{label}</h2>
        <p style={{ margin: 0, color: 'var(--text3)', fontSize: 12.5, lineHeight: 1.5 }}>{description}</p>
      </div>
      {items.length > 0 && (
        <div style={{ display: 'grid', gap: 8 }}>
          {items.map((item, index) => {
            const inputId = `${itemName}-${index + 1}`;
            const field = multiline ? (
              <textarea
                id={inputId}
                value={item}
                onChange={(event) => updateItem(index, event.target.value)}
                className="ac-input"
                rows={3}
                style={{ ...adminField, height: 'auto', minHeight: 76, padding: 10, lineHeight: 1.5, resize: 'vertical' }}
              />
            ) : (
              <input
                id={inputId}
                value={item}
                onChange={(event) => updateItem(index, event.target.value)}
                className="ac-input"
                spellCheck={false}
                style={{ ...adminField, fontFamily: 'var(--font-mono)' }}
              />
            );

            return (
              <div key={index} style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) auto', gap: 8, alignItems: 'end' }}>
                <label htmlFor={inputId} style={{ display: 'grid', gap: 6, minWidth: 0, fontSize: 12, fontWeight: 650 }}>
                  {label.slice(0, -1)} {index + 1}
                  {field}
                </label>
                <button
                  type="button"
                  aria-label={`Remove ${itemName} ${index + 1}`}
                  title={`Remove ${itemName} ${index + 1}`}
                  onClick={() => onChange(items.filter((_, currentIndex) => currentIndex !== index))}
                  className="ac-hover-surface2"
                  style={{ ...buttonStyles.iconButton(36), color: 'var(--text2)' }}
                >
                  <AdminIcon.X size={16} />
                </button>
              </div>
            );
          })}
        </div>
      )}
      <button
        type="button"
        onClick={() => onChange([...items, ''])}
        className="ac-hover-surface2"
        style={{ ...buttonStyles.secondary(36), marginTop: items.length ? 10 : 0 }}
      >
        <AdminIcon.Plus size={15} />
        Add {itemName}
      </button>
    </section>
  );
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
  const [tags, setTags] = useState<Tag[]>([]);
  const [tagsLoading, setTagsLoading] = useState(true);
  const [tagsError, setTagsError] = useState('');
  const [saving, setSaving] = useState(false);
  const savingRef = useRef(false);

  const loadTags = useCallback((signal?: AbortSignal) => {
    setTagsLoading(true);
    problemApi
      .tags(signal)
      .then((response) => {
        setTags(response.items ?? []);
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
                    color: 'var(--text)',
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
                </label>
              );
            })}
          </div>
        )}
      </section>

      <StringListField
        label="Constraints"
        description="Add one short technical condition per row."
        items={values.constraints}
        onChange={(constraints) => setValue('constraints', constraints)}
      />

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

      <StringListField
        label="Hints"
        description="Optional guidance for solvers. Each hint is kept as its own item."
        items={values.hints}
        onChange={(hints) => setValue('hints', hints)}
        multiline
      />

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
