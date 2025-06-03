# IPFIX Protocol Support Implementation Plan for Flowgre

## Overview
This plan outlines the steps to add IPFIX protocol support to Flowgre, building on its existing Netflow v9 implementation. IPFIX (RFC 7011) shares similarities with Netflow v9 but has key differences in template handling and field definitions that require targeted implementation.

## Technical Analysis
### Protocol Differences
1. **Header Version**: IPFIX uses version 10 vs Netflow v9's version 9
2. **Enterprise ID**: IPFIX requires enterprise-specific field IDs (IANA enterprise number)
3. **Template Management**: IPFIX has more flexible template handling with optional scope fields
4. **Data Format**: IPFIX allows variable-length fields and more standardized field definitions

## Implementation Steps

### 1. Code Structure Preparation
- [ ] Create new package `internal/protocol/ipfix`
- [ ] Define IPFIX-specific data structures mirroring `internal/protocol/netflow_v9`
- [ ] Implement IPFIX header parsing with version validation
- [ ] Add enterprise ID handling in template records

### 2. Collector Enhancements
- [ ] Modify UDP listener in `internal/collector` to detect IPFIX version
- [ ] Add protocol-specific handler registration in config
- [ ] Implement template cache with enterprise ID support
- [ ] Update metrics collection for IPFIX-specific counters

### 3. Processor Modifications
- [ ] Extend flow processor to handle IPFIX template records
- [ ] Add enterprise field mapping in data processing
- [ ] Implement variable-length field handling
- [ ] Update flow record validation logic

### 4. Configuration Management
- [ ] Add IPFIX-specific config options in `internal/config`
- [ ] Update viper configuration with protocol-specific timeouts
- [ ] Add enterprise ID whitelisting capability
- [ ] Update environment variable schema for IPFIX settings

### 5. Testing Strategy
- [ ] Add unit tests for IPFIX header parsing
- [ ] Create template record test suite with enterprise fields
- [ ] Implement integration tests with sample IPFIX packets
- [ ] Add protocol conformance checks in existing flow tests

### 6. Documentation Updates
- [ ] Update README with IPFIX configuration examples
- [ ] Add protocol comparison matrix in documentation
- [ ] Update metrics documentation for IPFIX counters
- [ ] Add enterprise ID configuration guide

## Dependencies
- [ ] Ensure Go 1.21+ for improved binary handling
- [ ] Verify expvar metrics package compatibility
- [ ] Check for any required updates to dependency packages

## Review Checklist
- [ ] Protocol version detection logic
- [ ] Enterprise ID handling security considerations
- [ ] Template cache memory management
- [ ] Backward compatibility with existing Netflow v9
- [ ] Performance impact assessment
- [ ] Metrics completeness for monitoring

## Next Steps
1. Review and finalize this plan
2. Implement protocol structures and parser
3. Integrate with collector and processor
4. Develop comprehensive test suite
5. Update documentation
6. Perform end-to-end validation