use crate::features::format_duration;
use compiler_schema::{CanonicalAnalysis, Event, LlmOutput};
use serde::Serialize;
use time::{OffsetDateTime, UtcOffset};

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct LlmContext {
    pub expr: Option<String>,
    pub expr_description: Option<String>,
}

pub fn project_output(analysis: &CanonicalAnalysis, context: &LlmContext) -> LlmOutput {
    let summary = build_summary(analysis);
    let analyze_result = AnalyzeResultPayload {
        summary,
        top_events: analysis
            .top_events
            .iter()
            .map(|event| event.kind.as_str())
            .collect(),
        event_time_summary: analysis
            .top_events
            .iter()
            .map(|event| build_event_time_summary(analysis, event))
            .collect(),
        peer_context: analysis
            .peer_context
            .as_ref()
            .filter(|peer_context| peer_context.total > 1)
            .map(|peer_context| PeerContextPayload {
                rank: peer_context.rank,
                total: peer_context.total,
                percentile: peer_context.percentile,
            }),
        evidence: analysis
            .evidence
            .iter()
            .take(3)
            .map(|item| EvidencePayload {
                label: item.label.as_str(),
                value: item.value.as_str(),
            })
            .collect(),
    };

    let analyze_result = serde_json::to_value(&analyze_result).unwrap_or_else(|err| {
        serde_json::json!({
            "error": format!("failed to encode analyze_result: {err}")
        })
    });

    LlmOutput {
        expr: non_empty_owned(context.expr.as_deref()),
        expr_description: non_empty_owned(context.expr_description.as_deref()),
        analyze_result,
    }
}

#[derive(Serialize)]
struct AnalyzeResultPayload<'a> {
    summary: SummaryPayload<'a>,
    top_events: Vec<&'a str>,
    event_time_summary: Vec<EventTimeSummaryPayload>,
    #[serde(skip_serializing_if = "Option::is_none")]
    peer_context: Option<PeerContextPayload>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    evidence: Vec<EvidencePayload<'a>>,
}

#[derive(Serialize)]
struct SummaryPayload<'a> {
    text: String,
    window: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    subject_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    metric_id: Option<String>,
    state: &'a str,
    trend: &'a str,
    problem_time: ProblemTimePayload,
}

#[derive(Serialize)]
struct ProblemTimePayload {
    mode: String,
    start: String,
    end: String,
    duration: String,
    display: String,
}

#[derive(Serialize)]
struct EventTimeSummaryPayload {
    kind: String,
    mode: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    count: Option<usize>,
    #[serde(skip_serializing_if = "Option::is_none")]
    first: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    last: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    start: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    end: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    duration: Option<String>,
    display: String,
}

#[derive(Serialize)]
struct PeerContextPayload {
    rank: usize,
    total: usize,
    percentile: usize,
}

#[derive(Serialize)]
struct EvidencePayload<'a> {
    label: &'a str,
    value: &'a str,
}

fn build_summary(analysis: &CanonicalAnalysis) -> SummaryPayload<'_> {
    let problem_time = build_problem_time(analysis);
    let summary_subject_id = sanitize_description_identifier(&analysis.subject_id);
    let summary_metric_id = sanitize_description_identifier(&analysis.metric_id);
    let subject_phrase = summary_subject_id
        .as_deref()
        .map(str::to_owned)
        .unwrap_or_else(|| default_subject_phrase(analysis.scope));
    let metric_phrase = match summary_metric_id.as_deref() {
        Some(metric_id) if Some(metric_id) != summary_subject_id.as_deref() => {
            format!(" on {metric_id}")
        }
        _ => String::new(),
    };
    SummaryPayload {
        text: format!(
            "In the {} window, {}{} was {} with a {} trend. Problem period: {}.",
            format_duration(analysis.window_secs.max(0)),
            subject_phrase,
            metric_phrase,
            humanize_token(analysis.state.as_str()),
            humanize_token(analysis.trend.as_str()),
            problem_time.display
        ),
        window: format_duration(analysis.window_secs.max(0)),
        subject_id: summary_subject_id,
        metric_id: summary_metric_id,
        state: analysis.state.as_str(),
        trend: analysis.trend.as_str(),
        problem_time,
    }
}

fn build_problem_time(analysis: &CanonicalAnalysis) -> ProblemTimePayload {
    let (start, end) = infer_problem_range_bounds(analysis);
    if let Some(window_start_ts_secs) = analysis.window_start_ts_secs {
        let absolute_start = window_start_ts_secs + start;
        let absolute_end = window_start_ts_secs + end;
        return ProblemTimePayload {
            mode: "absolute_range".into(),
            start: format_absolute_timestamp(absolute_start),
            end: format_absolute_timestamp(absolute_end),
            duration: format_duration((end - start).max(0)),
            display: format_absolute_range(absolute_start, absolute_end),
        };
    }

    ProblemTimePayload {
        mode: "relative_range".into(),
        start: format_offset(start),
        end: format_offset(end),
        duration: format_duration((end - start).max(0)),
        display: format!("{}->{}", format_offset(start), format_offset(end)),
    }
}

fn build_event_time_summary(
    analysis: &CanonicalAnalysis,
    event: &Event,
) -> EventTimeSummaryPayload {
    let kind = event.kind.as_str().to_string();
    if let Some(window_start_ts_secs) = analysis.window_start_ts_secs {
        if !event.timepoints_ts_secs.is_empty() {
            return build_absolute_points_summary(kind, window_start_ts_secs, &event.timepoints_ts_secs);
        }
        if event.start_ts_secs < event.end_ts_secs {
            return build_absolute_range_summary(kind, window_start_ts_secs, event.start_ts_secs, event.end_ts_secs);
        }
        return build_absolute_point_summary(kind, window_start_ts_secs + event.start_ts_secs);
    }

    if !event.timepoints_ts_secs.is_empty() {
        return build_relative_points_summary(kind, &event.timepoints_ts_secs);
    }
    if event.start_ts_secs < event.end_ts_secs {
        return build_relative_range_summary(kind, event.start_ts_secs, event.end_ts_secs);
    }
    build_relative_point_summary(kind, event.start_ts_secs)
}

fn build_absolute_points_summary(
    kind: String,
    window_start_ts_secs: i64,
    timepoints_ts_secs: &[i64],
) -> EventTimeSummaryPayload {
    let first_ts_secs = *timepoints_ts_secs.first().unwrap_or(&0);
    let last_ts_secs = *timepoints_ts_secs.last().unwrap_or(&first_ts_secs);
    let first = format_absolute_timestamp(window_start_ts_secs + first_ts_secs);
    let last = format_absolute_timestamp(window_start_ts_secs + last_ts_secs);
    let count = timepoints_ts_secs.len();
    let mode = if count <= 1 { "point" } else { "points" };
    let display = if count <= 1 {
        format!("{kind} at {first}")
    } else {
        format!("{kind}: count={count}, first={first}, last={last}")
    };

    EventTimeSummaryPayload {
        kind,
        mode: mode.into(),
        count: (count > 1).then_some(count),
        first: (count > 1).then_some(first.clone()),
        last: (count > 1).then_some(last.clone()),
        at: (count <= 1).then_some(first),
        start: None,
        end: None,
        duration: None,
        display,
    }
}

fn build_relative_points_summary(kind: String, timepoints_ts_secs: &[i64]) -> EventTimeSummaryPayload {
    let first_ts_secs = *timepoints_ts_secs.first().unwrap_or(&0);
    let last_ts_secs = *timepoints_ts_secs.last().unwrap_or(&first_ts_secs);
    let first = format_offset(first_ts_secs);
    let last = format_offset(last_ts_secs);
    let count = timepoints_ts_secs.len();
    let mode = if count <= 1 { "point" } else { "points" };
    let display = if count <= 1 {
        format!("{kind} at {first}")
    } else {
        format!("{kind}: count={count}, first={first}, last={last}")
    };

    EventTimeSummaryPayload {
        kind,
        mode: mode.into(),
        count: (count > 1).then_some(count),
        first: (count > 1).then_some(first.clone()),
        last: (count > 1).then_some(last.clone()),
        at: (count <= 1).then_some(first),
        start: None,
        end: None,
        duration: None,
        display,
    }
}

fn build_absolute_range_summary(
    kind: String,
    window_start_ts_secs: i64,
    start_ts_secs: i64,
    end_ts_secs: i64,
) -> EventTimeSummaryPayload {
    let start = format_absolute_timestamp(window_start_ts_secs + start_ts_secs);
    let end = format_absolute_timestamp(window_start_ts_secs + end_ts_secs);
    let duration = format_duration((end_ts_secs - start_ts_secs).max(0));
    let display = format!("{kind}: start={start}, end={end}, duration={duration}");

    EventTimeSummaryPayload {
        kind,
        mode: "range".into(),
        count: None,
        first: None,
        last: None,
        at: None,
        start: Some(start),
        end: Some(end),
        duration: Some(duration),
        display,
    }
}

fn build_relative_range_summary(
    kind: String,
    start_ts_secs: i64,
    end_ts_secs: i64,
) -> EventTimeSummaryPayload {
    let start = format_offset(start_ts_secs);
    let end = format_offset(end_ts_secs);
    let duration = format_duration((end_ts_secs - start_ts_secs).max(0));
    let display = format!("{kind}: start={start}, end={end}, duration={duration}");

    EventTimeSummaryPayload {
        kind,
        mode: "range".into(),
        count: None,
        first: None,
        last: None,
        at: None,
        start: Some(start),
        end: Some(end),
        duration: Some(duration),
        display,
    }
}

fn build_absolute_point_summary(kind: String, ts_secs: i64) -> EventTimeSummaryPayload {
    let at = format_absolute_timestamp(ts_secs);
    let display = format!("{kind} at {at}");
    EventTimeSummaryPayload {
        kind,
        mode: "point".into(),
        count: None,
        first: None,
        last: None,
        at: Some(at),
        start: None,
        end: None,
        duration: None,
        display,
    }
}

fn build_relative_point_summary(kind: String, ts_secs: i64) -> EventTimeSummaryPayload {
    let at = format_offset(ts_secs);
    let display = format!("{kind} at {at}");
    EventTimeSummaryPayload {
        kind,
        mode: "point".into(),
        count: None,
        first: None,
        last: None,
        at: Some(at),
        start: None,
        end: None,
        duration: None,
        display,
    }
}

fn infer_problem_range_bounds(analysis: &CanonicalAnalysis) -> (i64, i64) {
    let focused_events: Vec<_> = analysis
        .top_events
        .iter()
        .filter(|event| event.start_ts_secs < event.end_ts_secs)
        .filter(|event| {
            !matches!(
                event.kind,
                compiler_schema::EventKind::IncreasingTrend
                    | compiler_schema::EventKind::DecreasingTrend
            )
        })
        .collect();

    if let Some((start, end)) = focused_events
        .iter()
        .map(|event| (event.start_ts_secs, event.end_ts_secs))
        .reduce(|(left_start, left_end), (right_start, right_end)| {
            (left_start.min(right_start), left_end.max(right_end))
        })
    {
        return (start, end);
    }

    if let Some(event) = analysis.top_events.first() {
        return (event.start_ts_secs, event.end_ts_secs);
    }

    (0, analysis.window_secs.max(0))
}

fn format_offset(ts_secs: i64) -> String {
    let ts_secs = ts_secs.max(0);
    if ts_secs == 0 {
        "0m".into()
    } else {
        format_duration(ts_secs)
    }
}

fn format_absolute_range(start_ts_secs: i64, end_ts_secs: i64) -> String {
    let Ok(start) = OffsetDateTime::from_unix_timestamp(start_ts_secs) else {
        return format!("{}s->{}s", start_ts_secs, end_ts_secs);
    };
    let Ok(end) = OffsetDateTime::from_unix_timestamp(end_ts_secs) else {
        return format!("{}s->{}s", start_ts_secs, end_ts_secs);
    };
    let start = start.to_offset(UtcOffset::UTC);
    let end = end.to_offset(UtcOffset::UTC);

    if start.date() == end.date() {
        return format!(
            "{:04}-{:02}-{:02} {:02}:{:02}->{:02}:{:02} UTC",
            start.year(),
            u8::from(start.month()),
            start.day(),
            start.hour(),
            start.minute(),
            end.hour(),
            end.minute()
        );
    }

    format!(
        "{:04}-{:02}-{:02} {:02}:{:02} UTC->{:04}-{:02}-{:02} {:02}:{:02} UTC",
        start.year(),
        u8::from(start.month()),
        start.day(),
        start.hour(),
        start.minute(),
        end.year(),
        u8::from(end.month()),
        end.day(),
        end.hour(),
        end.minute()
    )
}

fn format_absolute_timestamp(ts_secs: i64) -> String {
    let Ok(point) = OffsetDateTime::from_unix_timestamp(ts_secs) else {
        return format!("{ts_secs}s");
    };
    let point = point.to_offset(UtcOffset::UTC);
    format!(
        "{:04}-{:02}-{:02} {:02}:{:02} UTC",
        point.year(),
        u8::from(point.month()),
        point.day(),
        point.hour(),
        point.minute()
    )
}

fn humanize_token(value: &str) -> String {
    value.replace('_', " ")
}

fn non_empty_owned(value: Option<&str>) -> Option<String> {
    let trimmed = value.unwrap_or("").trim();
    if trimmed.is_empty() {
        return None;
    }
    Some(trimmed.to_string())
}

fn sanitize_description_identifier(value: &str) -> Option<String> {
    let trimmed = value.trim();
    if trimmed.is_empty() || looks_like_query_expression(trimmed) {
        return None;
    }
    Some(trimmed.to_string())
}

fn default_subject_phrase(scope: compiler_schema::Scope) -> String {
    match scope {
        compiler_schema::Scope::Group => "this group".into(),
        compiler_schema::Scope::Line => "this series".into(),
    }
}

fn looks_like_query_expression(value: &str) -> bool {
    let operator_markers = [" by (", " without (", "[", "]", "{", "}", "/", "*", "+", "-", "sum(", "avg(", "max(", "min(", "rate(", "irate(", "histogram_quantile(", "increase("];
    if value.contains(' ') {
        return true;
    }
    if operator_markers.iter().any(|marker| value.contains(marker)) {
        return true;
    }
    value.contains('(') && value.contains(')')
}

#[cfg(test)]
mod tests {
    use super::{LlmContext, project_output};
    use compiler_schema::{
        CanonicalAnalysis, Event, EventKind, EvidenceItem, Scope, StateKind, TrendKind,
    };

    #[test]
    fn projection_uses_json_description_field() {
        let analysis = CanonicalAnalysis {
            schema_version: "v1",
            scope: Scope::Line,
            metric_id: "metric".into(),
            subject_id: "subject".into(),
            window_start_ts_secs: Some(1_774_951_200),
            window_secs: 600,
            state: StateKind::Elevated,
            trend: TrendKind::UpThenFlat,
            top_events: vec![Event {
                kind: EventKind::SustainedHigh,
                score: 8.0,
                start_ts_secs: 0,
                end_ts_secs: 600,
                timepoints_ts_secs: vec![],
                evidence: vec![EvidenceItem {
                    label: "duration".into(),
                    value: "10m".into(),
                }],
                impacted_members: 1,
            }],
            peer_context: None,
            regimes: vec![],
            evidence: vec![EvidenceItem {
                label: "duration".into(),
                value: "10m".into(),
            }],
        };

        let output = project_output(&analysis, &LlmContext::default());
        let decoded = &output.analyze_result;

        assert_eq!(decoded["summary"]["window"], "10m");
        assert_eq!(decoded["summary"]["subject_id"], "subject");
        assert_eq!(decoded["summary"]["metric_id"], "metric");
        assert_eq!(decoded["summary"]["state"], "elevated");
        assert_eq!(decoded["summary"]["trend"], "up_then_flat");
        assert_eq!(
            decoded["summary"]["problem_time"]["display"],
            "2026-03-31 10:00->10:10 UTC"
        );
        assert_eq!(decoded["top_events"], serde_json::json!(["sustained_high"]));
        assert_eq!(decoded["event_time_summary"][0]["mode"], "range");
        assert_eq!(
            decoded["event_time_summary"][0]["display"],
            "sustained_high: start=2026-03-31 10:00 UTC, end=2026-03-31 10:10 UTC, duration=10m"
        );
        assert_eq!(decoded["evidence"][0]["label"], "duration");
        assert_eq!(decoded["evidence"][0]["value"], "10m");
    }

    #[test]
    fn projection_hides_raw_query_identifiers_from_description_summary() {
        let query = "histogram_quantile(0.999, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le, instance))";
        let analysis = CanonicalAnalysis {
            schema_version: "v1",
            scope: Scope::Group,
            metric_id: query.into(),
            subject_id: query.into(),
            window_start_ts_secs: Some(1_776_641_040),
            window_secs: 600,
            state: StateKind::Stable,
            trend: TrendKind::Flat,
            top_events: vec![Event {
                kind: EventKind::Spike,
                score: 8.0,
                start_ts_secs: 0,
                end_ts_secs: 600,
                timepoints_ts_secs: vec![0, 60, 120],
                evidence: vec![],
                impacted_members: 3,
            }],
            peer_context: None,
            regimes: vec![],
            evidence: vec![],
        };

        let output = project_output(&analysis, &LlmContext::default());
        let decoded = &output.analyze_result;

        let summary_text = decoded["summary"]["text"]
            .as_str()
            .expect("summary text should be present");
        assert!(summary_text.contains("In the 10m window, this group was stable with a flat trend."));
        assert!(!summary_text.contains(query));
        assert!(decoded["summary"].get("subject_id").is_none());
        assert!(decoded["summary"].get("metric_id").is_none());
    }

    #[test]
    fn projection_includes_expr_metadata_when_provided() {
        let analysis = CanonicalAnalysis {
            schema_version: "v1",
            scope: Scope::Line,
            metric_id: "metric".into(),
            subject_id: "subject".into(),
            window_start_ts_secs: Some(1_774_951_200),
            window_secs: 600,
            state: StateKind::Stable,
            trend: TrendKind::Flat,
            top_events: vec![],
            peer_context: None,
            regimes: vec![],
            evidence: vec![],
        };

        let output = project_output(
            &analysis,
            &LlmContext {
                expr: Some("sum(rate(metric[1m]))".into()),
                expr_description: Some("per-instance rate".into()),
            },
        );

        assert_eq!(output.expr.as_deref(), Some("sum(rate(metric[1m]))"));
        assert_eq!(output.expr_description.as_deref(), Some("per-instance rate"));
    }
}
